//go:build linux
// +build linux

/*********************************************************************
 * Copyright (c) Intel Corporation 2021
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/
package heci

import (
	"bytes"
	"encoding/binary"
	"log"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

type Driver struct {
	meiDevice       *os.File
	bufferSize      uint32
	protocolVersion uint8
}

const (
	Device                   = "/dev/mei0"
	IOCTL_MEI_CONNECT_CLIENT = 0xC0104801
)

var MEI_IAMTHIF = [16]byte{0x28, 0x00, 0xf8, 0x12, 0xb7, 0xb4, 0x2d, 0x4b, 0xac, 0xa8, 0x46, 0xe0, 0xff, 0x65, 0x81, 0x4c}
var MEI_LMEIF = [16]byte{0xdb, 0xa4, 0x33, 0x67, 0x76, 0x04, 0x7b, 0x4e, 0xb3, 0xaf, 0xbc, 0xfc, 0x29, 0xbe, 0xe7, 0xa7}

// uint8 == uchar
type UUID_LE struct {
	uuid [16]uint8
}

func NewDriver() *Driver {
	return &Driver{}
}

func (heci *Driver) Init(useLME bool) error {

	var err error
	heci.meiDevice, err = os.OpenFile(Device, syscall.O_RDWR, 0)
	if err != nil {
		log.Println("Cannot open MEI Device")
		return err
	}

	data := CMEIConnectClientData{}
	if useLME {
		data.data = MEI_LMEIF
	} else {
		data.data = MEI_IAMTHIF
	}
	err = Ioctl(heci.meiDevice.Fd(), IOCTL_MEI_CONNECT_CLIENT, uintptr(unsafe.Pointer(&data)))
	if err != nil {
		return err
	}
	t := MEIConnectClientData{}
	err = binary.Read(bytes.NewBuffer(data.data[:]), binary.LittleEndian, &t)
	if err != nil {
		return err
	}

	heci.bufferSize = t.MaxMessageLength
	heci.protocolVersion = t.ProtocolVersion //should be 4?

	return nil
}
func (heci *Driver) GetBufferSize() uint32 {
	return heci.bufferSize
}
func (heci *Driver) SendMessage(buffer []byte, done *uint32) (bytesWritten uint32, err error) {

	size, err := syscall.Write(int(heci.meiDevice.Fd()), buffer)
	if err != nil {
		return 0, err
	}

	return uint32(size), nil
}
func (heci *Driver) ReceiveMessage(buffer []byte, done *uint32) (bytesRead uint32, err error) {

	read, err := unix.Read(int(heci.meiDevice.Fd()), buffer)
	if err != nil {
		return 0, err
	}
	return uint32(read), nil
}

func Ioctl(fd, op, arg uintptr) error {
	_, _, ep := syscall.Syscall(syscall.SYS_IOCTL, fd, op, arg)
	if ep != 0 {
		return syscall.Errno(ep)
	}
	return nil
}

func (heci *Driver) Close() {
	defer heci.meiDevice.Close()
}