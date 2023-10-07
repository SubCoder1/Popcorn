# Popcorn <img src=https://github.com/SubCoder1/Popcorn/assets/40127554/f2c453a0-1096-45f2-99ac-532a183aca9c width="80">

![Visual Studio Code](https://img.shields.io/badge/Visual%20Studio%20Code-0078d7.svg?style=for-the-badge&logo=visual-studio-code&logoColor=white)
![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Redis](https://img.shields.io/badge/redis-%23DD0031.svg?style=for-the-badge&logo=redis&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)

Popcorn is live! at https://popcorntv.click 


Let's say you're very far from your friends and family, but you want to watch a movie together. With Popcorn, you can create a virtual space
where everyone can join, hangout and watch anything with each other! This repository features the server side of Popcorn, while the 
front end is presented by [Popcorn-web](https://github.com/SubCoder1/Popcorn-web).

## Architecture
<img width="1920" alt="Popcorn architecture (1)" src="https://github.com/SubCoder1/Popcorn/assets/40127554/48b0e1ea-eeb8-4dc9-951a-c11a94720a08">

## Requirements (Without Docker)

**Go** <1.20 or above>

**Redis** <5 or above>

**Node** <20 or above>

**NPM** <9 or above>

**VueJS** <3 or above>

## Installation (With Docker)

1. Get [Livekit](https://livekit.io/) Host, API, Secret and RTMP Host credentials and save those in ```config/secrets.env```. This is a one time thing.

2. Create a docker network using the command below:
   ```console
   docker network create -d bridge popcorn-network
   ``` 

4. Clone this repository and run it using the command below:
   
   ```console
   docker compose --env-file=./config/secrets.env up --build
   ```

5. Clone [Popcorn-web](https://github.com/SubCoder1/Popcorn-web) and run it using the command below:

    ```console
    docker compose up --build
    ```
6. Launch the nginx docker container, which'll receive both the backend and the frontend's traffic:

   ```console
   // In Popcorn repository
   cd nginx/
   
   docker compose -f nginx-compose.yaml up --build 
   ```
7. Finally, Open http://localhost and try it out!

## Installation (without Docker)

1. Get [Livekit](https://livekit.io/) Host, API, Secret and RTMP Host credentials and save those in ```config/secrets.env```. This is a one time thing.

2. Clone this repository and run it using the command below (Make sure redis-server is installed):

   ```console
   go mod download

   make load-db

   make local

   # Server is running on port 8080
   ```

4. Clone [Popcorn-web](https://github.com/SubCoder1/Popcorn-web).
5. Inside Popcorn-web repository, Set ```VUE_APP_SERVER_URL = http://localhost:8080``` (line-1).
6. Run the frontend using the command below:
   ```console
      npm run serve -- --port 8081
   ```
8. Finally, Open http://localhost:8081 and try it out!   
