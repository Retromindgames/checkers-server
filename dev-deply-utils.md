### Docker

docker attach <container_id>

Or use an interactive shell:

docker exec -it <container_id> /bin/sh  # Use /bin/bash if available


docker logs --since 5m <container_id>


### Transfer Files to Server:

scp -r /home/ec2-user/ssl/* user@remote_host:/home/ec2-user/ssl/
