# Popcorn ![logo-128x128](https://github.com/SubCoder1/Popcorn/assets/40127554/f2c453a0-1096-45f2-99ac-532a183aca9c)

![Visual Studio Code](https://img.shields.io/badge/Visual%20Studio%20Code-0078d7.svg?style=for-the-badge&logo=visual-studio-code&logoColor=white)
![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Redis](https://img.shields.io/badge/redis-%23DD0031.svg?style=for-the-badge&logo=redis&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)

Lets assume you're very far away from your friends & family, but you'll want to watch a movie together. 
Popcorn lets you create a virtual room where everyone can join, chat and watch movie together! 
This repository presents the server side of Popcorn, while the frontend is presented by [Popcorn-web](https://github.com/SubCoder1/Popcorn-web).


## Requirements (Without Docker)

**Go** <1.20 or above>

**Redis** <5 or above>

**Node** <20 or above>

**NPM** <9 or above>

**VueJS** <3 or above>

## Installation (With Docker)

1. Get [Livekit](https://livekit.io/) Host, API, Secret and RTMP Host credentials and save those in ```config/secrets.env```. This is a one time thing.

2. Clone this repository and run it using the command below:
   
   ```console
   docker compose --env-file=./config/secrets.env up --build
   ```
3. Launch the nginx docker container, which'll receive both the backend and the frontend's traffic:

   ```console
   cd nginx/
   
   docker compose -f nginx-compose.yaml up --build 
   ```

4. Clone [Popcorn-web](https://github.com/SubCoder1/Popcorn-web) and run it using the command below:

    ```console
    docker compose up --build
    ```
5. Finally, Open http://localhost and try it out!

## Installation (without Docker)

1. Get [Livekit](https://livekit.io/) Host, API, Secret and RTMP Host credentials and save those in ```config/secrets.env```. This is a one time thing.

2. Clone this repository and run it using the command below (Make sure redis-server is installed):

   ```console
   go mod download

   make load-db

   make local
   ```

4. Clone [Popcorn-web](https://github.com/SubCoder1/Popcorn-web) and run it using the command below:

   ```console
   npm run serve -- --port 8081
   ```
5. Finally, Open http://localhost:8081 and try it out!   
