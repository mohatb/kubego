# kubego
kubego is a rewritten go implementation of kubectl-exec https://github.com/mohatb/kubectl-exec

> :warning: **Windows SSH is still not supported but will be added soon**

## Installation:

### Install on Linux:

```bash
wget https://github.com/mohatb/kubego/raw/master/bin/kubego-amd64-linux
chmod +x kubego-amd64-linux
sudo mv kubego-amd64-linux /usr/local/bin/kubego
kubego
```


### Install on Windows:

```powershell
Open Powershell
Invoke-WebRequest -Uri "https://github.com/mohatb/kubego/raw/master/bin/kubego-amd64-windows.exe" -OutFile "$($env:userprofile)\AppData\Local\Microsoft\WindowsApps\kubego.exe"
kubego.exe
```

### Install on Azure Cloud Shell:

Since Azure cloud shell does not allow sudo, you can run this as a script.
```bash
wget https://github.com/mohatb/kubego/raw/master/bin/kubego-amd64-linux
chmod +x kubego-amd64-linux
mv kubego-amd64-linux kubego
./kubego
```
