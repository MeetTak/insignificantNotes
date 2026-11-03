---
title: "sandbox instructions"
folder: "aws"
created: "2025-10-30T20:07:19.000Z"
modified: "2025-10-30T20:07:48.000Z"
note_id: "x-coredata://8788A586-C8FC-4484-AB68-760AF85E0359/ICNote/p2859"
source: "apple-notes"
---

**AWS Academy Cloud Foundations – Sandbox**
**Environment Overview**
This sandbox provides an environment for ad-hoc exploration of AWS services.
**This environment is cleaned up at the end of every session**. When the session timer runs to 0:00, the session will end, and any data and resources that you created in the AWS account will be deleted.
**Region restriction**
All service access is limited to the **us-east-1** Region. If you load a service console page in another AWS Region you will see access error messages.
**Service usage and other restrictions**
The following services can be used. Specific limitations apply as documented. Services restrictions are subject to change.

- AWS Cloud9
  - Supported Instance types: nano, micro, and small.
- AWS CloudFormation
- Amazon CloudFront
- AWS CloudShell
- AWS CloudTrail
- Amazon CloudWatch
- Amazon EC2 Auto Scaling
  - Supported Instance types: t2.nano, t2.micro, t2.small, t3.nano, t3.micro, and t3.small
  - Read the *Concurrently running instances limits* details documented in the EC2 service details below to be aware of further restrictions.
  - *Recommendation*: size to your actual need to avoid using up your cost budget.
- AWS Elastic Beanstalk
  - *To create an application*: choose **Create Application**, give it an application name, choose a platform, then choose **Configure more options**. Scroll down to the Security panel and choose **Edit**. For *Service role*, choose **LabRole**. If the environment is in the us-east-1 AWS Region, for *EC2 key pair,* choose **vockey**. For *IAM instance profile*, choose **LabInstanceProfile**. Choose **Save**, then choose **Create app**.
  - Supported Instance types: nano, micro, small, medium, and large. If you attempt to launch a larger instance type, it will be terminated.
- Amazon Elastic Compute Cloud (Amazon EC2)
  - Amazon Machine Images (AMIs) – You can use Amazon provided Linux and Microsoft Windows AMIs
  - Instances – You can only launch the following instance types: t2.nano, t2.micro, t2.small, t2.medium,t3.nano, t3.micro, t3.small and t3.medium
  - Amazon Elastic Block Store (Amazon EBS) volumes – You can use volumes in sizes up to 35 GB and of type *General Purpose SSD* (gp2)
  - On-Demand Instances only
  - Maximum of 9 concurrently running EC2 instances, regardless of instance size. If you attempt to launch more, the excess instances will be terminated (and 9 will be left running).
  - A role named **LabRole** and an instance profile named **LabInstanceProfile** have been pre-created for you. You can attach the role (via the instance profile) to an EC2 instance when you want to access an EC2 instance (terminal in the browser) using AWS Systems Manager Session Manager. The role also grants permissions to any applications running on the instance to access many other AWS services from the instance.
- Elastic Load Balancing
- AWS Identity and Access Management (IAM)
  - Read-only access.
  - A role named **LabRole** has been pre-created for you. This role is designed to be used when you want to attach a role to a resource in an AWS service. It grants many AWS services access to other AWS services and has permissions very similar to the permissions you have as a user in the console.
  - You cannot create roles, except that you can create service-linked roles.
- AWS Key Management Service (AWS KMS) – You have list access
- AWS Lambda
  - When creating a function, change the default role and use the existing **LabRole**.
- Amazon Relational Database Service (Amazon RDS)
  - Supported database engines: Amazon Aurora, MySQL, PostgreSQL and MariaDB.
  - Multi-AZ deployments are not supported (choose *Dev/Test* or *Free tier* template if prompted and do not create a standby instance).
  - Supported instance types: nano, micro, small, and medium (Tip: choose *Burstable classes* to find these).
  - Supported storage types: EBS volumes - size up to 100 GB and type General Purpose SSD (gp2). PIOPS storage types are not supported.
  - On-Demand DB instance class types only.
  - ***Enhanced monitoring is not supported*** (you must *uncheck* this default setting in the *Additional configuration / Monitoring* panel).
- Amazon Route 53
  - You cannot register a domain
- Amazon Simple Notification Service (Amazon SNS)
- Amazon Simple Storage Service (Amazon S3)
  - You cannot create a vault lock
- AWS Systems Manager (SSM)
  - A role named **LabRole** and an instance profile named **LabInstanceProfile** have been pre-created for you. You can attach the role (via the instance profile) to an EC2 instance when you want to access an EC2 instance (terminal in the browser) using AWS Systems Manager Session Manager.
- Tagging

**Accessing the AWS Management Console**

1. At the top of these instructions, choose Start Lab to launch your lab. A **Start Lab** panel opens, and it displays the lab status.
2. Wait until you see the message *Lab status: ready*, then close the **Start Lab** panel by choosing the **X**.
3. At the top of these instructions, choose AWS This opens the AWS Management Console in a new browser tab. The system will automatically log you in. **Tip**: If a new browser tab does not open, a banner or icon is usually at the top of your browser with the message that your browser is preventing the site from opening pop-up windows. Choose the banner or icon, and then choose **Allow pop-ups**.
4. Arrange the AWS Management Console tab so that it displays alongside these instructions. Ideally, you will have both browser tabs open at the same time so that you can follow the lab steps more easily.

**Using the terminal in the browser**
A terminal window displays to the right of these instructions.
You can toggle the visibility of the terminal window by selecting or clearing the check box in the *Terminal* box at the top of the screen.
The terminal in the browser provides access to a Linux shell on a server that exists *outside* of the AWS account that you use when your lab is running.
**Running AWS CLI commands**
After you start the lab, the terminal will be preconfigured with the credentials that you need to use the AWS Command Line Interface (AWS CLI).
For example, to retrieve the account number and your user ID, run the following command:

```
aws sts get-caller-identity
```

If you have any EC2 instances that are running in the sandbox, this command provides information about them:

```
aws ec2 describe-instances
```

For details about how to use the AWS CLI, refer to the <u>[AWS CLI command reference](https://docs.aws.amazon.com/cli/latest/reference/)</u> documentation.
**Using the AWS SDK for Python**
Python V3 is installed in the terminal, with access to the AWS software development kit (SDK) for Python (Boto3) library. You can use this library to run AWS SDK for Python code. For example:

```
$ python3
>>> import boto3
>>> ec2 = boto3.client('ec2', region_name='us-east-1')
>>> ec2.describe_regions()
>>> exit()
$
```

For details about how to use the AWS SDK for Python, refer to the <u>[Boto3 documentation](https://boto3.amazonaws.com/v1/documentation/api/latest/index.html)</u> page.
**Accessing EC2 instances**
When you launch EC2 instances in the sandbox environment, choose the option to use the existing key pair that's named *vockey* at the time of launch. Then:

- Above these instructions (that you are currently reading), choose the Details dropdown menu.
- Choose Show A **Credentials** window opens.
  - If you use a *Microsoft Windows computer* and want to connect to a Linux instance, or connect to a Microsoft Windows instance by using Secure Shell (SSH): Choose **Download PPK** and save the **labsuser.ppk** file.
  - If you want to connect to a Windows instance desktop from any computer: Choose **Download PEM** and save the **labsuser.pem** file.
  - If you use *macOS* and want to connect to a Linux instance by using SSH: Choose **Download PEM** and save the **labsuser.pem** file.
- Typically, your browser will save the labsuser.pem or the labsuser.ppk file to your **Downloads** directory.
- Exit the **Credentials** window by choosing the **X**.
- *To connect to the desktop of a Microsoft Windows instance*:
  - In the EC2 console, choose **Instances** and then choose the instance that you want to connect to.
  - From the **Actions** menu, choose **Get Windows Password**.
  - Next to *Key Pair Path*, choose **Browse**.
  - Browse to and select the labsuser.pem file that you downloaded earlier.
  - Choose **Decrypt Password**.
  - The connection information should display, including the instance's public Domain Name System (DNS), administrator user name, and the decrypted password.
  - Use a Remote Desktop Protocol (RDP) client to connect to the desktop of the EC2 instance by using these connection details.
- *To connect to a Linux instance by using SSH, see the next section*.

**Using SSH to access any EC2 instances that you launch**
The following steps describe how to use the SSH key to connect to your instance.
**Microsoft Windows users**
These instructions are for Windows users only.
If you use macOS or Linux, <u>[skip to the next section](https://labs.vocareum.com/web/4547493/4540316.0/ASNLIB/public/docs/lang/en-us/README.html?vockey=a00c0bc3c0198068a8eeed972c61f6879b2ec9941b34e0a43b79f1f8afb9554e#ssh-MACLinux)</u>.

1. Download the required software.
   - You will use *PuTTY* to use SSH to connect to EC2 instances. If you do not have PuTTY installed on your computer, <u>[download it here](https://the.earth.li/~sgtatham/putty/latest/w64/putty.exe)</u>.
2. Open **putty.exe**
3. Configure PuTTY to not timeout:
   - Choose **Connection**
   - Set **Seconds between keepalives** to 30
4. This setting enables you to keep the PuTTY session open for a longer time.
5. Configure your PuTTY session:
   - Choose **Session**
   - **Host Name (or IP address):** Copy and paste the **IPv4 Public IP address** for the instance. To find it, return to the Amazon EC2 console and choose **Instances**. Select the box next to the instance and in the **Description** tab, copy the **IPv4 Public IP** value.
   - Back in PuTTY, in the **Connection** list, expand **SSH**
   - Choose **Auth** and expand **Credentials**
   - Under **Private key file for authentication:** Choose **Browse**
   - Browse to the *labsuser.ppk* file that you downloaded, select it, and choose **Open**
   - Choose **Open** again
6. To trust and connect to the host, choose **Accept**.
7. When prompted to **login as**, enter: ec2-user This step will connect you to the EC2 instance.
8. <u>[Microsoft Windows users: Click here to skip ahead to the next task.](https://labs.vocareum.com/web/4547493/4540316.0/ASNLIB/public/docs/lang/en-us/README.html?vockey=a00c0bc3c0198068a8eeed972c61f6879b2ec9941b34e0a43b79f1f8afb9554e#ssh-after)</u>

**macOS and Linux Users - Using SSH to Connect**
These instructions are only for macOS or Linux users.

1. Read through the three bullet points in this step before you start to complete the actions. You will not be able see these instructions when the **Details** panel is open.
   - Above these instructions you are currently reading, choose the Details dropdown menu, and then choose Show
2. A **Credentials** window will open.
   - Choose the **Download PEM** button and save the **labsuser.pem** file. Typically, your browser will save it to the **Downloads** directory.
   - Then, exit the Details panel by choosing the **X**.
3. Open a terminal window, and change directory (cd) to the directory where the .pem file was downloaded. For example, if it was saved to your **Downloads** directory, run this command:       cd \~/Downloads
4. Change the permissions on the key to be *read only* by running this command:       chmod 400 \<filename>.pem
5. Return to the AWS Management Console, and in the EC2 service, choose **Instances**. Check the box next to the instance you want to connect to.
6. In the **Description** tab, copy the **IPv4 Public IP** value.
7. Return to the terminal window and run this command (replace **\<*public-ip-address*>** with the actual public IP address that you copied):       ssh -i \<filename>.pem ec2-user@\<public-ip-address>
8. When you are prompted to allow a first connection to this remote SSH server, enter yes. Because you are using a key pair for authentication, you will not be prompted for a password.
