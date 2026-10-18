## To build image
1. Install docker and login to your docker account
2. Download files
3. In `Dockerfile` replace `<token>` with your bot's token and save file
4. In directory run `docker build -t <username>/boxes .`
5. Run `docker push <username>/boxes`

To run this app on your server: 
 - Install docker. 
 - Pull images: `mysql` and `<username>/boxes` from Docker Hub. 
 - Create a network `boxes` for this app. 
 - Run `mysql` container by command `docker run -d --network boxes-net --name mysql -e MYSQL_ROOT_PASSWORD=<your_pswd> -e MYSQL_DATABASE=boxes mysql` 
 - Run `<username>/boxes` container by commant `docker run -d --network boxes-net --name boxes -e MYSQL_PASSWORD=<your_pswd> <username>/boxes`
> [!note]
> Before using these commands replace
>  - `<your_pswd>` with password you want to use to acces to DB
>  - `<username>` with your Docker username
>  - `<token>` with telegram-bot's token
