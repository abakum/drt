//go:build !windows

package main

func mkLink(oldname, newname string, link, hard bool) (err error) {
	return
}
