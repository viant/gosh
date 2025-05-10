# gosh - Local/remove shell executor

This library is compatible with Go 1.20+

Please refer to [`CHANGELOG.md`](CHANGELOG.md) if you encounter breaking changes.


- [Motivation](#motivation)
- [Usage](#usage)
- [License](#license)
- [Credits and Acknowledgements](#credits-and-acknowledgements)



## Motivation


### Enhancing Efficiency and Productivity
The primary motivation for developing a local remote shell is to significantly enhance the efficiency and productivity of developers, system administrators, and IT professionals. By allowing users to execute commands on remote systems without the need to manually log into each system, the tool reduces time and effort spent on routine tasks. This efficiency is particularly valuable in environments where managing multiple servers or systems is a regular requirement.
### Simplifying Complex Operations
Another key motivation is to simplify complex operations that involve multiple steps or commands to be executed on remote systems. The local remote shell can be designed to support the execution of scripted commands or sequences of operations, thereby abstracting the complexity and reducing the potential for errors. This simplification is crucial for maintaining system integrity and ensuring consistent configurations across distributed environments.
### Supporting Scalability and Flexibility
Finally, the local remote shell project is motivated by the need for a scalable and flexible tool that can adapt to diverse environments and requirements. Whether it is managing a handful of servers or scaling to enterprise-level infrastructure, the tool will be designed to handle varying loads efficiently. Moreover, the flexibility to integrate with existing workflows and support for extension or customization will make it a valuable asset in any technology stack.

In conclusion, the motivation behind developing a local remote shell using os.Exec is driven by the desire to create a powerful, efficient, and user-friendly tool that addresses the practical challenges of managing remote systems. By focusing on efficiency, simplicity, security, and scalability, the project aims to deliver a solution that enhances productivity and simplifies remote system management for a wide range of users.



## Usage

### Local Host
```go
package main

import (
	"github.com/viant/gosh"
	"github.com/viant/gosh/local"
	"context
)


func ExampleLocalRun() {
	ctx := context.Background()
    srv, err := gosh.New(ctx, local.New())
    if err != nil {
    return
    }
    _, _, err = srv.Run(ctx, "cd /etc")
    output, _, err := srv.Run(ctx, "ls -l")
    println(output)
}
```


### Remove Host
```go
package main

import (
	"github.com/viant/gosh"
	"github.com/viant/gosh/local"
	"github.com/viant/scy/cred"
	"github.com/viant/gosh/runner/ssh"
	"context"
	"os"

)


func ExampleRemoveRun() {
	host := "remote-host:22"
	sshCred := &cred.SSH{
		PrivateKeyLocation:"/path/to/your/private/key",
		Basic: cred.Basic{
			Username: os.Getenv("USER"),
		},
	}
	clientConfig, err := sshCred.Config(context.Background())
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	srv, err := gosh.New(ctx, ssh.New(host+":22", clientConfig))
	if err != nil {
		return
	}
	_, _, err = srv.Run(ctx, "cd /etc")
	output, _, err := srv.Run(ctx, "ls -l")
	println(output)
}

```
## Model Context Protocol Integration

The `gosh` library can be integrated with the Model Context Protocol (MCP) to provide a seamless experience for executing commands in a  local or remote shell environment. The integration allows for efficient communication between the client and server, enabling real-time command execution and response handling.

```go
package mypackage

import (
	"context"
	serverproto "github.com/viant/mcp-protocol/server"
	"github.com/viant/mcp-protocol/schema"
	"github.com/viant/mcp/server"
	"github.com/viant/gosh/mcp"

)	


func ExampleMCPIntegration() error {

	newImplementer := serverproto.WithDefaultImplementer(context.Background(), func(implementer *serverproto.DefaultImplementer) error {
		err := mcp.Register(implementer)
		return err
	})
	srv, err := server.New(
        server.WithNewImplementer(newImplementer),
        server.WithImplementation(schema.Implementation{"default", "1.0"}),
        server.WithCapabilities(schema.ServerCapabilities{Resources: &schema.ServerCapabilitiesResources{}}),
    )
    if err != nil {
		return  err
    }
	return srv.HTTP(context.Background(), ":4981").ListenAndServe()
}
    
```


## License

The source code is made available under the terms of the Apache License, Version 2, as stated in the file `LICENSE`.

Individual files may be made available under their own specific license,
all compatible with Apache License, Version 2. Please see individual files for details.

## Credits and Acknowledgements

Authors:

- Adrian Witas
