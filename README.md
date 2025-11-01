# RESTree

Simple CLI tool to recursively build `.http` file in a given directory

## Why

I was tired of using memory-hungry tools like Insomnia or Postman for simple API calls. I wanted something lightweight that I could easily edit with my favorite editor. So I designed this minimalist tool for those who prefer terminal workflows.


## Installation

### Manual installation

**Prerequisites**: `git`, `make`, `go`

To manually install the program, follow these steps:

```sh
git clone https://github.com/kamil-koziol/restree
cd restree
make install
```

This will build the project and install the restree binary to your system.

### Installation using AUR

You can install [restree-git](https://aur.archlinux.org/packages/restree-git) from the [Arch User Repository (AUR)](https://aur.archlinux.org).

### Installation using Go

Alternatively, you can install the `latest` version of restree directly using Go:

```sh
go install https://github.com/kamil-koziol/restree@latest
```

## Use case

### Testing API endpoint

You can easily test your API endpoints by using `restree` in combination with a simple tool like [http2curl](https://github.com/kamil-koziol/http2curl) to convert the `.http` file into curl command that you can later run in your shell


**Example:**

```sh
restree build users/get.http | http2curl | bash
```

## Simple Guide

Given the following directory structure:

```sh
.
├── _headers.http
└── users
    ├── _headers.http
    └── get.http
```

You can define headers in any `_headers.http` file:
```
// ./_headers.http

Authorization: Basic {{auth}}
Content-Type: application/json
```

```
// ./users/_headers.http

User-Header: "cooluser"
```

`RESTree` collects and merges headers from all `_headers.http` files from the root to the target file.

Your request file could look like this:
```
// ./users/get.http

GET {{host}}/users
```

Then in your shell:
```
# Set the variables
export auth=$(echo "username:password" | base64)
export host="http://localhost/users"

restree build users/get.http
```

This produces:
```
GET http://localhost/users
Authorization: Basic
Content-Type: application/json
User-Header: cooluser
```

### Running scripts before request

You can define `_before.sh` file that will be ran before the `_headers.http` files is handled.

```
.
└── users
    ├── _before.sh
    ├── _headers.http
    └── get.http
```

And it can be used to set the `env` variables:


```bash
# _before.sh

echo "variable=gucamole"
```

That will be later used in header files and the final request itselft:

```
// ./_headers.http

Header: {{variable}}
```
