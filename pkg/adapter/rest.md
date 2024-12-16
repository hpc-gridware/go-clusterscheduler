# REST API

## Adapter

Adapter is an experimental internal qconf to REST converter allowing
to execute `qconf` commands through http POST calls.

## Canonical Example of a qconf REST API

This is another example of exposing qconf through a REST API.

### Cluster Configuration

- **GET** `/cluster/configuration`: Get the cluster configuration
- **POST** `/cluster/configuration`: Apply the cluster configuration

### Calendar Operations

- **POST** `/calendars`: Add a new calendar
- **DELETE** `/calendars/{calendarName}`: Delete a calendar
- **GET** `/calendars/{calendarName}`: Show a specific calendar
- **GET** `/calendars`: Show all calendars
- **PUT** `/calendars/{calendarName}`: Modify a specific calendar

### Complex Entries

- **POST** `/complexentries`: Add a complex entry
- **DELETE** `/complexentries/{entryName}`: Delete a complex entry
- **GET** `/complexentries/{entryName}`: Show a specific complex entry
- **GET** `/complexentries`: Show all complex entries
- **GET** `/complexes`: Show all complexes
- **PUT** `/complexes`: Modify all complexes

### Checkpoint Interfaces

- **POST** `/ckptinterfaces`: Add a checkpoint interface
- **DELETE** `/ckptinterfaces/{interfaceName}`: Delete a checkpoint interface
- **GET** `/ckptinterfaces/{interfaceName}`: Show a specific checkpoint interface
- **GET** `/ckptinterfaces`: Show all checkpoint interfaces
- **PUT** `/ckptinterfaces/{interfaceName}`: Modify a specific checkpoint interface

### Host Config

- **POST** `/hostconfigurations`: Add host configuration
- **DELETE** `/hostconfigurations/{configName}`: Delete host configuration
- **GET** `/hostconfigurations/{hostName}`: Show a specific host configuration
- **GET** `/hostconfigurations`: Show all host configurations
- **PUT** `/hostconfigurations/{configName}`: Modify a specific host configuration

### Global Configuration

- **GET** `/globalconfiguration`: Show global configuration
- **PUT** `/globalconfiguration`: Modify global configuration

### Execution Hosts

- **POST** `/executionhosts`: Add an execution host
- **DELETE** `/executionhosts/{hostList}`: Delete execution hosts
- **PUT** `/executionhosts/{execHostName}`: Modify a specific execution host
- **GET** `/executionhosts/{hostName}`: Show a specific execution host
- **GET** `/executionhosts`: Show all execution hosts

### Admin Hosts

- **POST** `/adminhosts`: Add administrative hosts
- **DELETE** `/adminhosts`: Delete administrative hosts
- **GET** `/adminhosts`: Show all administrative hosts

### Host Groups

- **POST** `/hostgroups`: Add a host group
- **PUT** `/hostgroups/{hostGroupName}`: Modify a specific host group
- **DELETE** `/hostgroups/{groupName}`: Delete a host group
- **GET** `/hostgroups/{groupName}`: Show a specific host group
- **GET** `/hostgroups`: Show all host groups

### Resource Quota Sets

- **POST** `/resourcequotasets`: Add a resource quota set
- **DELETE** `/resourcequotasets/{rqsList}`: Delete a resource quota set
- **GET** `/resourcequotasets/{rqsList}`: Show a specific resource quota set
- **GET** `/resourcequotasets`: Show all resource quota sets
- **PUT** `/resourcequotasets/{rqsName}`: Modify a specific resource quota set

### Manager List Operations

- **POST** `/managers`: Add users to manager list
- **DELETE** `/managers`: Delete users from manager list
- **GET** `/managers`: Show all managers

### Operator List Operations

- **POST** `/operators`: Add users to operator list
- **DELETE** `/operators`: Delete users from operator list
- **GET** `/operators`: Show all operators

### Parallel Environments

- **POST** `/parallelenvironments`: Add a parallel environment
- **DELETE** `/parallelenvironments/{peName}`: Delete a parallel environment
- **GET** `/parallelenvironments/{peName}`: Show a specific parallel environment
- **GET** `/parallelenvironments`: Show all parallel environments
- **PUT** `/parallelenvironments/{peName}`: Modify a specific parallel environment

### Projects

- **POST** `/projects`: Add a project
- **DELETE** `/projects`: Delete projects
- **GET** `/projects/{projectName}`: Show a specific project
- **GET** `/projects`: Show all projects
- **PUT** `/projects/{projectName}`: Modify a specific project

### Cluster Queues

- **POST** `/clusterqueues`: Add a cluster queue
- **PUT** `/clusterqueues/{queueName}`: Modify a specific cluster queue
- **DELETE** `/clusterqueues/{queueName}`: Delete a cluster queue
- **GET** `/clusterqueues/{queueName}`: Show a specific cluster queue
- **GET** `/clusterqueues`: Show all cluster queues

### Submit Hosts

- **POST** `/submithosts`: Add submit hosts
- **DELETE** `/submithosts`: Delete submit hosts
- **GET** `/submithosts`: Show all submit hosts

### Share Tree Operations

- **PUT** `/sharetree/nodes`: Modify share tree nodes
- **DELETE** `/sharetree/nodes`: Delete share tree nodes
- **GET** `/sharetree/nodes`: Show share tree nodes
- **GET** `/sharetree`: Show the share tree
- **PUT** `/sharetree`: Modify the share tree
- **DELETE** `/sharetree/usage`: Clear share tree usage

### User Sets

- **POST** `/usersets/{listnameList}`: Add a user set list
- **POST** `/usersets/{listnameList}/users`: Add users to a user set list
- **DELETE** `/usersets/{listnameList}/users`: Delete users from a user set list
- **DELETE** `/usersets/{listnameList}`: Delete a user set list
- **GET** `/usersets/{listnameList}`: Show a specific user set list
- **GET** `/usersets`: Show all user set lists
- **PUT** `/usersets/{listnameList}`: Modify a specific user set

### Users

- **POST** `/users`: Add a user
- **DELETE** `/users`: Delete users
- **GET** `/users/{userName}`: Show a specific user
- **GET** `/users`: Show all users
- **PUT** `/users/{userName}`: Modify a specific user

### Miscellaneous Operations

- **DELETE** `/usage`: Clear sharetree usage
- **DELETE** `/queue/clean`: Clean specific queues
- **PUT** `/execdaemons/shutdown`: Shutdown execution daemons
- **PUT** `/daemon/master/shutdown`: Shutdown the master daemon
- **PUT** `/daemon/scheduling/shutdown`: Shutdown the scheduling daemon
- **DELETE** `/eventclients`: Kill event clients
- **DELETE** `/threads/qmaster`: Kill qmaster thread
- **PUT** `/attributes/{objectName}`: Modify an attribute
- **DELETE** `/attributes/{objectName}`: Delete an attribute
- **POST** `/attributes/{objectName}`: Add an attribute
- **PUT** `/schedulerconfig`: Modify the scheduler configuration
- **GET** `/schedulerconfig`: Show the scheduler configuration
