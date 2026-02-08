# add labels to a node
```shell
kubectl label node worker-a-0 topology.kubernetes.io/zone=az-a --overwrite
```


# ubi-image-get-infos

Initialize a module

In the directory where your main.go lives:

go mod init mymodule


Replace mymodule with whatever module name you want (often a GitHub URL):

go mod init github.com/username/projectname


This command creates:

go.mod

✅ 2. Automatically generate go.sum

After initializing the module, run:

go mod tidy


This does two things:

Resolves and downloads dependencies

Generates go.sum

After this you will have both:

go.mod
go.sum

Example Directory
project/
 ├── main.go
 ├── go.mod      <-- generated
 └── go.sum      <-- generated