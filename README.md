# solid-mon-rpi

`solid-mon-rpi` is an HTTP server publishing data from `I2C` and `SMBus` sensors.

## Example

```sh
$> solid-mon-rpi -addr=:80 -cfg=./config.xml
solid-mon-rpi starting up web-server on: :80
solid-mon-rpi cfg: [{Name:Temperature sensor 1 ChanID:3 Type:AT30TSE} {Name:Humidity sensor 1 ChanID:1 Type:HTS221} {Name:Onboard sensors ChanID:7 Type:Onboard}]
[...]
```

### client

One can inspect what `solid-mon-rpi` serves like so:

```sh
$> curl clrmedaq01.in2p3.fr:80/echo
{"timestamp":"2017-06-21T14:34:19.551842601Z","sensors":[{"name":"Temperature sensor 1","type":"temperature","value":30},{"name":"Humidity sensor 1","type":"humidity","value":41.65479908390589},{"name":"Humidity sensor 1","type":"temperature","value":31.226401179941004},{"name":"Onboard sensors","type":"pressure","value":968.3974435752888},{"name":"Onboard sensors","type":"luminosity","value":183.76320000000004}],"labels":{"Humidity sensor 1":["humidity","temperature"],"Onboard sensors":["pressure","luminosity"],"Temperature sensor 1":["temperature"]}}
```

## Installation on a new RPi

### Binary installation

Download binary (for your architecture) from github:

- https://github.com/sbinet-solid/solid-mon-rpi/releases/download/v0.1/solid-mon-rpi-linux-amd64.exe
- https://github.com/sbinet-solid/solid-mon-rpi/releases/download/v0.1/solid-mon-rpi-linux-arm.exe

*(Don't forget to download the latest release! The links above are for v0.1)*

and then use the `deploy` script:

```sh
$> git clone https://github.com/sbinet-solid/solid-mon-rpi
$> cd solid-mon-rpi
$> curl -O -L https://github.com/sbinet-solid/solid-mon-rpi/releases/download/v0.1/solid-mon-rpi-linux-arm.exe
$> chmod +x ./solid-mon-rpi-linux-arm.exe
$> ./deploy me@example.com ./solid-mon-rpi-linux-arm.exe
```

### Installation from source

- Install the [Go](https://golang.org) SDK for your platform from https://golang.org/dl
- Install the `solid-mon-rpi` binary:

```sh
$> go get github.com/sbinet-solid/solid-mon-rpi
```

and then, cross-compile for `ARM` and deploy:

```sh
$> cd $GOPATH/src/github.com/sbinet-solid/solid-mon-rpi
$> ./build-deploy me@example.com
```
