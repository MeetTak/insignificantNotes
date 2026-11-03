---
title: "sonatype nexus installation"
folder: "cloud"
created: "2025-12-24T18:06:51.000Z"
modified: "2025-12-26T10:14:44.000Z"
note_id: "x-coredata://8788A586-C8FC-4484-AB68-760AF85E0359/ICNote/p3306"
source: "apple-notes"
---

enter into ur cloud machine with the required credentials
enter into ur /opt directory
use the wget command to install nexus
use the tar command

```
tar -zxvf nexus-3.87.1-01-linux-x86_64.tar.gz
```

two new folders will be created

- nexus folder

it contains runtime and application of nexus

- sonatype-work folder

it contains own config for nexus and data

create a new user with the name nexus(or u can choose as per ur choice)
use the chown command to set the user and group as nexus instead of root

```
chown -R nexus:nexus sonatype-work
chown -R nexus:nexus nexus-3.87.1-01
```

under  /opt/nexus-3.87.1-01/bin/nexus.rc
add this line
run_as_user="nexus”

to start the nexus service use this command
/opt/nexus-3.87.1-01/bin/nexus start

to check if its running
ps aux | grep nexus

netstat -lnpt

check the port and open it in the web browser accordingly

check for the password for admin as the user
here at this location to login
/opt/sonatype-work/nexus3/admin.password
