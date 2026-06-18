# Incubator

Incubator is a lightweight virtual machine manager written in Go on top of QEMU/KVM.

The project was built as a learning platform to understand virtualization internals without relying on higher-level tooling such as libvirt, Proxmox, OpenStack, or LXD.

## Features

* Create virtual machines from cloud images
* Launch VMs with cloud-init support
* Start existing VMs
* Stop running VMs
* Destroy stopped VMs
* List managed VMs
* Persistent VM metadata storage
* Automatic disk provisioning using qcow2 backing images
* VNC console access

## Design Goals

* Minimal dependencies
* Direct interaction with QEMU
* Simple CLI workflow
* Learn virtualization fundamentals
* Avoid abstraction layers until necessary

## Current Architecture

```text
Incubator
    ↓
QEMU
    ↓
KVM
    ↓
Linux
```

VM metadata is stored locally and used to track:

* Resource name
* VM status
* CPU allocation
* Memory allocation
* Disk configuration
* Operating system image
* Process ID
* VNC port assignment

## Example

```bash
incubator launch debian

incubator list

incubator stop clever_tesla

incubator start clever_tesla

incubator destroy clever_tesla
```

## Future Work

* SQLite metadata backend
* QMP integration
* Snapshot management
* TAP networking
* Open vSwitch integration
* OVN-backed networking
* Reconciliation loop
* Multi-node orchestration

## Disclaimer

Incubator is currently an educational project and should not be considered a production virtualization platform.

Its primary purpose is to explore how modern VM platforms are built from first principles.
