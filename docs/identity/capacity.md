# Capacity Module
The capacity module is responsible for collection information about the node capacity (amount of resources it has and what it can do). It uses this information as well to make sure the node is registered on the grid.

## Registration process
Capacity daemon starts after the node identity has been generated hence it waits for `identityd` to be fully up. Once node keys are created (or loaded) it does the following:

- The module collects all the required information for registration this include:
  - CPU
  - Memory
  - Amount of SSD Storage
  - Amount of HDD storage
  - Public Network Configuration
- Capacity daemon then loads node private key from identity module
- It makes sure an account associated with that key already exists on grid-db
  - If not exist, one is created using an activation service.
- Capacity daemon then tries to find a twin object associated with its account.
  - If not exist a twin object is created with the node yggdrasil IPv6 address
  - If a twin exists, we make sure it has the correct IP, otherwise a call to update it is done.
- Capacity daemon then tried to find a node object associated with its account.
  - If not exist a node object is created with the right capacity information gathered.
  - If a node exists, we make sure it has the correct capacity information, otherwise an update call is done.
- Once node registration is complete, capacity module makes sure an instance of the `msgbus` is running with the correct twin ID. The process is monitored and auto spawned if the twin (msgbus) dies for any reason.
- Capacity daemon still does 2 extra things after registration is complete:
  - Send uptime information to grid-db (currently every 8 hours)
  - Provide internal endpoints for monitoring (some information that shows up on the summary screen)
