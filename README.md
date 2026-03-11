# HYV
Remote Monitoring and Management (RMM) Platform

# TODO
- [ ] Database first-time setup
- [ ] Building Agent/Updater
- [X] Mac Agent
    - [X] Runs as service
    - [X] Collects basic inventory
    - [X] Executes remote commands
    - [X] Streams system performance data
- [ ] Windows Agent
    - [ ] Runs as service
    - [ ] Collects basic inventory
    - [ ] Executes remote commands

# Environment Variables
- For connecting to DB
  - PGSQL_USER='postgres'
  - PGSQL_PASS='somePaSSw0RD'
- For building agent and updater
  - HYV_CONTROL_SERVER_HOST='mydomain.com'
  - HYV_CONTROL_SERVER_PORT='2213'
- For authentication
  - PASS_ROUNDS=8
  - PASS_SALT='SomethingSalty'



