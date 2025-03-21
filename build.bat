rsrc -manifest EverCliping.manifest -ico ./assets/icon.ico -o EverCliping.syso
go build -v -a -ldflags "-s -w -H windowsgui" -gcflags="all=-trimpath=${PWD}" -asmflags="all=-trimpath=${PWD}" .
