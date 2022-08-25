# Web Cloud Infrastructure 

--- 

Prototype of the Cloud Infrastructure, that provides ability to setup Virtual Servers with Custom Configuration and OS. Currently is planned to
be using this Service commercially

--- 

# Usage 

If you want to run it on your own, `Application` + Cloud Compute Server 
You need to setup HostMachine, so it can communicate with the API. 
Folow this Guide to setup `Host Machine`, which going to store the VM's, Networks etc... 
Requirements ~ Hardware (Memory at least 5Gi) (CPU at least 1.5Ghz), so in order to comfortably run VM servers

Once you've setup Host Machine, you need to follow this steps 

*If you want to run Application on your Local Machine* 


---

### Requirements 

1. ~ Make Sure that the Ports `8000` and `3000` are open, otherwise that would fail with Exception 

2. ~ Make Sure that the `Docker` and `Docker-Compose` is installed on your local machine 

---


### Backend Build Steps 

1. Edit Project Environment Variable File located at Root Directory at Path Called `env`

2. Once you've done that, You can run docker-compose File and it will Run the App locally 
```
$ git clone https://github.com/LovePelmeni/Cloud-Infrastructure.git  # Cloning Project Repo 
$ cd ./docker-compose # cd to the directory with docker-compose file 
$ docker-compose up -d # run docker-compose file 
```

Great! Now You Successfully Run Backend Application, in order to Check if it's Up and Running, you can Execute: 

```
curl -X GET -f http://localhost:8000/ping/
```


### Frontend Build Steps 

1. Edit Project Environment Variable File located at Root Directory at Path Called `env`

2. Once you've done that, You can run docker-compose File and it will Run the Frontend App locally 
```
$ git clone https://github.com/LovePelmeni/Cloud-Infrastructure-Front-App.git  # Cloning Project Repo 
$ cd ./docker-compose # cd to the directory with docker-compose file 
$ docker-compose up -d # run docker-compose file 

```

Great! Now You Successfully Run Backend Application, in order to Check if it's Up and Running, you can Execute: 

```
curl -X GET -f http://localhost:3000/ping/
```

--- 

Great! Now you setup the Frontend Application for the Project and The Whole App is Fully Configured 

You can go to your Browser at "http://localhost:3000/" and it will redirect you to the Cloud Infrastructure App 


