package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/dolmen-go/contextio"
	"github.com/machinebox/progress"
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

	cioOut := contextio.NewWriter(o.ctx, local)
	prgOut := progress.NewWriter(cioOut)

	o.startProgress(o.currFile,
		o.totalFile,
		o.selindex,
		int64(entry.Size),
		prgOut,
		recursive)

	_, err = io.Copy(prgOut, remote)
	if err != nil {
		return err
	}

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
	prgIn := progress.NewReader(cioIn)

	o.startProgress(o.currFile,
		o.totalFile,
		o.selindex,
		entry.Size(),
		prgIn,
		recursive)

	_, err = io.Copy(remote, prgIn)

	if err != nil {
		return err
	}

	return nil
}

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
	prgIn := progress.NewReader(cioIn)

	o.startProgress(o.currFile,
		o.totalFile,
		o.selindex,
		entry.Size(),
		prgIn,
		recursive)

	_, err = io.Copy(dstFile, prgIn)
	if err != nil {
		return err
	}

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
	if o.totalFile > 0 {
		return nil
	}

	if o.transfer == adbToAdb || o.transfer == adbToLocal {
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

		return nil
	}

	err := filepath.Walk(src, func(p string, entry os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !entry.IsDir() {
			o.totalFile++
		}

		return nil
	})

	return err
}
