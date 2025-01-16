To run this app on your server: Install docker. 
Pull images: `mysql` and `danpik/boxes` from Docker Hub. 
Create a network `boxes` for this app. 
Run mysql container by command `docker run -d --network boxes --name mysql -e MYSQL_ROOT_PASSWORD=<your_pswd> -e MYSQL_DATABASE=boxes mysql` 
Run danpik/boxes container by commant `docker run -d --network boxes --name boxes -e MYSQL_PASSWORD=<your_pswd> danpik/boxwes`
> [!note]
> before using docker commands replace <your_pswd> to password you want to use to acces to DB
