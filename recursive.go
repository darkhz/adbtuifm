package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dolmen-go/contextio"
	"github.com/machinebox/progress"
	adb "github.com/zach-klippenstein/goadb"
)

func (o *opsWork) pushRecursive(src, dst string, device *adb.Device) error {
	select {
	case <-o.ctx.Done():
		return o.ctx.Err()
	default:
	}

	stat, err := os.Stat(src)
	if err != nil {
		o.opErr(statError)
		return err
	}

	fil, err := os.Open(src)
	if err != nil {
		o.opErr(openError)
		return err
	}
	defer fil.Close()

	_, err = device.RunCommand("mkdir " + dst)
	if err != nil {
		return err
	}

	mode := fmt.Sprintf("%04o", stat.Mode().Perm())
	_, err = device.RunCommand("chmod " + mode + " " + dst)
	if err != nil {
		return err
	}

	list, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range list {
		s := filepath.Join(src, entry.Name())
		d := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			o.pushRecursive(s, d, device)
			continue
		}

		perms := entry.Mode().Perm()
		mtime := entry.ModTime()

		local, err := os.Open(s)
		if err != nil {
			o.opErr(openError)
			return err
		}
		defer local.Close()

		remote, err := device.OpenWrite(d, perms, mtime)
		if err != nil {
			o.opErr(createError)
			return err
		}
		defer remote.Close()

		stat, _ := os.Stat(s)
		cioIn := contextio.NewReader(o.ctx, local)
		prgIn := progress.NewReader(cioIn)

		o.startProgress(o.currFile, stat.Size(), prgIn, true)

		_, err = io.Copy(remote, prgIn)

		if err != nil {
			return err
		}
	}

	return nil
}

func (o *opsWork) pullRecursive(src, dst string, device *adb.Device) error {
	select {
	case <-o.ctx.Done():
		return o.ctx.Err()
	default:
	}

	stat, err := device.Stat(src)
	if err != nil {
		o.opErr(statError)
		return err
	}

	if err = os.MkdirAll(dst, stat.Mode); err != nil {
		return err
	}

	list, err := device.ListDirEntries(src)

	for list.Next() {
		entry := list.Entry()
		name := entry.Name

		s := filepath.Join(src, name)
		d := filepath.Join(dst, name)

		if entry.Mode&os.ModeDir != 0 {
			err = o.pullRecursive(s, d, device)
			if err != nil {
				return err
			}
			continue
		}

		remote, err := device.OpenRead(s)
		if err != nil {
			o.opErr(createError)
			return err
		}
		defer remote.Close()

		local, err := os.Create(d)
		if err != nil {
			o.opErr(openError)
			return err
		}
		defer local.Close()

		cioOut := contextio.NewWriter(o.ctx, local)
		prgOut := progress.NewWriter(cioOut)

		o.startProgress(o.currFile, int64(entry.Size), prgOut, true)

		_, err = io.Copy(prgOut, remote)
		if err != nil {
			return err
		}
	}
	if list.Err() != nil {
		return err
	}

	return nil
}

func (o *opsWork) copyRecursive(src, dst string) error {
	select {
	case <-o.ctx.Done():
		return o.ctx.Err()
	default:
	}

	stat, err := os.Stat(src)
	if err != nil {
		o.opErr(statError)
		return err
	}

	fil, err := os.Open(src)
	if err != nil {
		o.opErr(openError)
		return err
	}
	defer fil.Close()

	if err := os.MkdirAll(dst, stat.Mode()); err != nil {
		return err
	}

	list, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range list {
		s := filepath.Join(src, entry.Name())
		d := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			o.copyRecursive(s, d)
			continue
		}

		dstFile, err := os.Create(d)
		if err != nil {
			o.opErr(createError)
			return err
		}
		defer dstFile.Close()

		srcFile, err := os.Open(s)
		if err != nil {
			o.opErr(openError)
			return err
		}
		defer srcFile.Close()

		stat, _ := os.Stat(s)
		cioIn := contextio.NewReader(o.ctx, srcFile)
		prgIn := progress.NewReader(cioIn)

		o.startProgress(o.currFile, stat.Size(), prgIn, true)

		_, err = io.Copy(dstFile, prgIn)
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *opsWork) getTotalFiles() error {
	if o.transfer == adbToAdb || o.transfer == adbToLocal {
		_, device := getAdb()
		if device == nil {
			return errors.New("No ADB device")
		}

		cmd := "find " + o.src + " -type f | wc -l"
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

	err := filepath.Walk(o.src, func(p string, entry os.FileInfo, err error) error {
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
