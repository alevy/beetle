# Beetle

Beetle is an operating system service that mediates access between applications
and Bluetooth Low Energy peripherals using GATT (Generic Attribute Profile),
the application-level protocol of Bluetooth Low Energy. While GATT was designed
for low-power personal area networks, its connection-oriented interface, naming
hierarchy, and transactional semantics give sufficient structure for an OS to
manage and understand application behavior and properly manage access to
peripherals without needing to understand device specific functionality.

## Building and Running

Beetle is built for Linux kernels compiled with the Bluez subsystem (most
desktop distributions, but not, for example, Andorid). The only compile-time
dependency is a recent Go compiler. There are no specific runtime dependencies,
but you need a way to turn on your Bluetooth controller and scan for
peripherals, so the Bluez userland tools (specifically `hciconfig` and
`hcitool` are useul).

```bash
$ cd PATH_TO_BEETLE_SOURCE
$ go build main.go
$ ./main
```

Next ensure your Bluetooth controller is powered on and scan for your peripherals:

```bash
$ sudo hciconfig hci0 up
$ sudo hcitool lescan
```

Running Beetle presents a shell interface with several [commands](#commands).
The following is an example session that connects to two BLE peripheral devices
and allows them to interact with each other:

```bash
$ ./main
> connect random [DEVICE1_ADDRESS]
Connecting to [DEVICE1_ADDRESS]
> connect random [DEVICE2_ADDRESS]
Connecting to [DEVICE2_ADDRESS]
> serve 0 1
> serve 1 0
> start 0
> start 1
```

## Commands

+------------+-------------------------------+---------------------------------+
| Command    | Arguments                     | Description                     |
+============+===============================+=================================+
| connect    | public|random DEVICE\_ADDRESS | Connects to a peripheral device.|
|            |                               | The first argument specifies if |
|            |                               | the address is Public or Random.|
+------------+-------------------------------+---------------------------------+
| connectTCP | IP:PORT                       | Connects to a remote TCP server.|
+------------+-------------------------------+---------------------------------+
| devices    |                               | Lists connected devices by      |
|            |                               | device number.                  |
+------------+-------------------------------+---------------------------------+
| start      | DEVICE\_NUM                   | Performs discovery on the device|
|            |                               | and begins communication with   |
|            |                               | it.                             |
+------------+-------------------------------+---------------------------------+
| start      | DEVICE\_NUM                   | Same as `start` but without     |
|            |                               | performing GATT discovery.      |
+------------+-------------------------------+---------------------------------+
| disconnect | DEVICE\_NUM                   | Disconnects from the specified  |
|            |                               | device.                         |
+------------+-------------------------------+---------------------------------+
| handles    | DEVICE\_NUM                   | Lists handles associated with   |
|            |                               | the device (discovered by the   |
|            |                               | `start` command).               |
+------------+-------------------------------+---------------------------------+
| serve      | DEVICE\_FROM DEVICE\_TO       | Exposes handles from            |
|            |                               | `DEVICE\_FROM` to `DEVICE\_TO`. |
+------------+-------------------------------+---------------------------------+
| debug      | on|off                        | Turns debugging (prints GATT    |
|            |                               | commands to the console) on or  |
|            |                               | off.                            |
+------------+-------------------------------+---------------------------------+

