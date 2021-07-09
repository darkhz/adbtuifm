package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/dolmen-go/contextio"
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

		cioIn := contextio.NewReader(o.ctx, local)

		_, err = io.Copy(remote, cioIn)
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

		_, err = io.Copy(cioOut, remote)
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

		cioIn := contextio.NewReader(o.ctx, srcFile)
		_, err = io.Copy(dstFile, cioIn)
		if err != nil {
			return err
		}
	}

	return nil
}
