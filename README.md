# RESTree

Simple CLI tool to recursively build `.http` file in a given directory

## Why

I was tired of using memory-hungry tools like Insomnia or Postman for simple API calls. I wanted something lightweight that I could easily edit with my favorite editor. So I designed this minimalist tool for those who prefer terminal workflows.


## Installation

```sh
go install https://github.com/kamil-koziol/restree@latest
```

## Guide

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

restree users/get.http
```

This produces:
```
GET http://localhost/users

Authorization: Basic
Content-Type: application/json
User-Header: cooluser
```

