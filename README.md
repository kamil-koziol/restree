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

You can easily test your API endpoints by using `restree run`.

**Example:**

```sh
restree run users/get.http
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

## Neovim integration

The following Lua snippet adds a `Restree` command that executes the request
from the current `.http` buffer and shows the response in a split window.

Optional flags:
- `jq` – pretty-prints JSON responses using `jq`
- `headers` – also displays request/response headers

> Note: the `jq` option requires `jq` to be installed.

Paste this snippet into your Neovim configuration:

```lua
local function run_restree(args)
  local filepath = vim.fn.expand("%:p")
  local show_headers = string.find(args, "headers") ~= nil
  local use_jq = string.find(args, "jq") ~= nil

  -- Create the output buffer
  vim.cmd("vsplit | wincmd l | enew")
  local buf = vim.api.nvim_get_current_buf()

  -- Buffer boilerplate
  vim.bo[buf].buftype = "nofile"
  vim.bo[buf].bufhidden = "wipe"
  vim.bo[buf].swapfile = false
  vim.keymap.set("n", "q", "<cmd>q<CR>", { buffer = buf, silent = true })

  -- Build the shell command string
  -- We use sh -c to handle the pipe to jq if needed
  local cmd_str = "restree run -k -v " .. vim.fn.shellescape(filepath)
  if use_jq then
    cmd_str = cmd_str .. " | jq"
    vim.bo[buf].filetype = "json"
  end

  vim.system({ "sh", "-c", cmd_str }, { text = true }, function(res)
    vim.schedule(function()
      if not vim.api.nvim_buf_is_valid(buf) then
        return
      end

      local lines = {}

      -- Handle Headers (from stderr)
      if show_headers and res.stderr and res.stderr ~= "" then
        vim.list_extend(lines, vim.split(res.stderr, "\n", { plain = true }))
        table.insert(lines, "") -- Spacer
      end

      if res.stdout and res.stdout ~= "" then
        vim.list_extend(lines, vim.split(res.stdout, "\n", { plain = true }))
      end

      -- Handle Errors (if exit code isn't 0 and we haven't shown stderr yet)
      if res.code ~= 0 and not show_headers then
        table.insert(lines, "--- ERROR (Exit Code " .. res.code .. ") ---")
        vim.list_extend(lines, vim.split(res.stderr or "Unknown Error", "\n", { plain = true }))
      end

      vim.bo[buf].modifiable = true
      vim.api.nvim_buf_set_lines(buf, 0, -1, false, lines)
      vim.bo[buf].modifiable = false
    end)
  end)
end

vim.api.nvim_create_user_command("Restree", function(opts)
  run_restree(opts.args)
end, { nargs = "*" })

-- Keybindings
vim.api.nvim_create_autocmd("FileType", {
  pattern = "http",
  callback = function()
    -- Quick run (Body only)
    vim.keymap.set("n", "<leader>rr", ":Restree<CR>", { buffer = true, silent = true })
    -- Run with JQ
    vim.keymap.set("n", "<leader>rj", ":Restree jq<CR>", { buffer = true, silent = true })
    -- Run with Headers + JQ
    vim.keymap.set("n", "<leader>ra", ":Restree jq headers<CR>", { buffer = true, silent = true })
  end,
})
```
