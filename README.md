# goCANFlasher

![goCANFlasher](gocanflasher.jpg)

Trionic 5/7/8 ecu flasher GUI made with [Fyne.io](https://fyne.io/)

## Starting

### Windows 

    $env:CGO_ENABLED=1; $env:GOOS="windows"; $env:GOARCH="386"; go run .

### Linux

    CGO_ENABLED=1 GOOS=linux go run .

## Todo

Add support for Trionic 8 (T8) flashing. Only dumping is currently available for this ECU.