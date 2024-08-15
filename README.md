To run this app on your server: 
Install `docker`.
Pull images: `mysql` and `danpik/boxes` from Docker Hub. 
Create a network for this app. 
Run mysql container by command `docker run -d --network <net_name> --network-alias mysql -e MYSQL_ROOT_PASSWORD=root -e MYSQL_DATABASE=boxes mysql`.
Run danpik/boxes container by commant `docker run -d --network <net_name> --name boxes danpik/boxes`
