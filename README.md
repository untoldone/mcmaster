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