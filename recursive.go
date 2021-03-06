package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/dolmen-go/contextio"
	"github.com/schollz/progressbar/v3"
	adb "github.com/zach-klippenstein/goadb"
)

func (o *operation) pullFile(src, dst string, entry *adb.DirEntry, device *adb.Device, recursive bool) error {
	remote, err := device.OpenRead(src)
	if err != nil {
		return err
	}
	defer remote.Close()

	local, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer local.Close()

	cioIn := contextio.NewReader(o.ctx, remote)
	prgIn := progressbar.NewReader(cioIn, o.progress.pbar)

	_, err = io.Copy(local, &prgIn)
	if err != nil {
		return err
	}

	o.updatePb()

	return nil
}

func (o *operation) pullRecursive(src, dst string, device *adb.Device) error {
	select {
	case <-o.ctx.Done():
		return o.ctx.Err()

	default:
	}

	if o.opmode != opCopy {
		return fmt.Errorf("%s not implemented via pull", o.opmode.String())
	}

	stat, err := device.Stat(src)
	if err != nil {
		return err
	}

	if !stat.Mode.IsDir() {
		return o.pullFile(src, dst, stat, device, false)
	}

	if err = os.MkdirAll(dst, stat.Mode); err != nil {
		return err
	}

	list, err := device.ListDirEntries(src)

	for list.Next() {
		entry := list.Entry()

		s := filepath.Join(src, entry.Name)
		d := filepath.Join(dst, entry.Name)

		if entry.Mode&os.ModeDir != 0 {
			if err = o.pullRecursive(s, d, device); err != nil {
				return err
			}
			continue
		}

		if err = o.pullFile(s, d, entry, device, true); err != nil {
			return err
		}
	}
	if list.Err() != nil {
		return err
	}

	return nil
}

func (o *operation) pushFile(src, dst string, entry os.FileInfo, device *adb.Device, recursive bool) error {
	var err error

	switch {
	case entry.Mode()&os.ModeSymlink != 0:
		src, err = filepath.EvalSymlinks(src)
		if err != nil {
			return err
		}

	case entry.Mode()&os.ModeNamedPipe != 0:
		return nil
	}

	mtime := entry.ModTime()
	perms := entry.Mode().Perm()

	local, err := os.Open(src)
	if err != nil {
		return err
	}
	defer local.Close()

	remote, err := device.OpenWrite(dst, perms, mtime)
	if err != nil {
		return err
	}
	defer remote.Close()

	cioIn := contextio.NewReader(o.ctx, local)
	prgIn := progressbar.NewReader(cioIn, o.progress.pbar)

	_, err = io.Copy(remote, &prgIn)
	if err != nil {
		return err
	}

	o.updatePb()

	return nil
}

//gocyclo:ignore
func (o *operation) pushRecursive(src, dst string, device *adb.Device) error {
	select {
	case <-o.ctx.Done():
		return o.ctx.Err()

	default:
	}

	if o.opmode != opCopy {
		return fmt.Errorf("%s not implemented via push", o.opmode.String())
	}

	stat, err := os.Lstat(src)
	if err != nil {
		return err
	}

	if !stat.Mode().IsDir() {
		return o.pushFile(src, dst, stat, device, false)
	}

	srcfd, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcfd.Close()

	cmd := fmt.Sprintf("mkdir '%s'", dst)
	out, err := device.RunCommand(cmd)
	if err != nil {
		return err
	} else if out != "" {
		return fmt.Errorf(out)
	}

	mode := fmt.Sprintf("%04o", stat.Mode().Perm())
	cmd = fmt.Sprintf("chmod %s '%s'", mode, dst)
	out, err = device.RunCommand(cmd)
	if err != nil {
		return err
	} else if out != "" {
		return fmt.Errorf(out)
	}

	list, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range list {
		s := filepath.Join(src, entry.Name())
		d := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err = o.pushRecursive(s, d, device); err != nil {
				return err
			}
			continue
		}

		if err = o.pushFile(s, d, entry, device, true); err != nil {
			return err
		}
	}

	return nil
}

func (o *operation) copyFile(src, dst string, entry os.FileInfo, recursive bool) error {
	var err error

	switch {
	case entry.Mode()&os.ModeSymlink != 0:
		src, err = filepath.EvalSymlinks(src)
		if err != nil {
			return err
		}

	case entry.Mode()&os.ModeNamedPipe != 0:
		return syscall.Mkfifo(dst, uint32(entry.Mode()))
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	cioIn := contextio.NewReader(o.ctx, srcFile)
	prgIn := progressbar.NewReader(cioIn, o.progress.pbar)

	_, err = io.Copy(dstFile, &prgIn)
	if err != nil {
		return err
	}

	o.updatePb()

	return nil
}

func (o *operation) copyRecursive(src, dst string) error {
	select {
	case <-o.ctx.Done():
		return o.ctx.Err()

	default:
	}

	var list []os.FileInfo

	stat, err := os.Lstat(src)
	if err != nil {
		return err
	}

	if !stat.Mode().IsDir() {
		return o.copyFile(src, dst, stat, false)
	}

	srcfd, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcfd.Close()

	if err := os.MkdirAll(dst, stat.Mode()); err != nil {
		return err
	}

	list, err = ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range list {
		s := filepath.Join(src, entry.Name())
		d := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err = o.copyRecursive(s, d); err != nil {
				return err
			}
			continue
		}

		if err = o.copyFile(s, d, entry, true); err != nil {
			return err
		}
	}

	return nil
}

func (o *operation) getTotalFiles(src string) error {
	if o.totalFile > 0 || o.opmode != opCopy {
		return nil
	}

	if o.transfer == adbToAdb {
		return nil
	}

	if o.transfer == adbToLocal {
		device, err := getAdb()
		if err != nil {
			return err
		}

		cmd := fmt.Sprintf("find '%s' -type f | wc -l", src)
		out, err := device.RunCommand(cmd)
		if err != nil {
			return err
		}

		o.totalFile, err = strconv.Atoi(strings.TrimSuffix(out, "\n"))
		if err != nil {
			return err
		}

		cmd = fmt.Sprintf("du -d0 -sh '%s'", src)
		out, err = device.RunCommand(cmd)
		if err != nil {
			return err
		}

		o.totalBytes, err = getByteSize(strings.Fields(out)[0])
		if err != nil {
			return err
		}

		return nil
	}

	err := filepath.Walk(src, func(p string, entry os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !entry.IsDir() {
			o.totalFile++
			o.totalBytes += entry.Size()
		}

		return nil
	})

	return err
}

func getByteSize(str string) (int64, error) {
	var exp int
	var err error
	var size int64

	const unit = 1024
	const suffixes = "KMGTPE"

	num := str[:len(str)-1]
	suffix := str[len(str)-1:]

	for i := 0; i < len(suffixes); i++ {
		if string(suffixes[i]) == suffix {
			exp = i
			break
		}
	}

	if strings.Contains(num, ".") {
		num = strings.Split(num, ".")[0]
	}

	size, err = strconv.ParseInt(num, 10, 64)
	if err != nil {
		return 0, err
	}

	return int64(size) * int64(math.Pow(unit, float64(exp+1))), nil
}
