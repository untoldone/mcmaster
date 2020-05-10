Dependencies to build / use dev environment:

* Go 1.13 or later
* Node 12.16.1 or later
* Java for running minecraft
* GNU make to run build scripts

To start dev environment:

    make run-dev

To build for production:

    make

To just install dependencies:

    make deps

Configuration:

Set environment variables to configure McMaster.

* USER_WHITELIST: Comma delimited list of minecraft email addresses allowed to log in
* MINECRAFT_CLIENT_TOKEN: Identifier token to the Minecraft authentication server. Can be anything you want
* HMAC_SECRET_KEY: Secret key used to create and verify JWT based client tokens. Keep this safe
* MINECRAFT_DIRECTORY: Directory where server.jar should exist. If missing, it will be downloaded. Will run minecraft from here