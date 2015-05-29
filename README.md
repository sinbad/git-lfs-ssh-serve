# The Git-LFS SSH reference server #

`git-lfs-ssh-serve` is a reference implementation of a pure SSH server for 
[git-lfs](https://github.com/github/git-lfs).

When using an SSH URL (either ssh://user@host/path or user@host:/path), git-lfs
will automatically open an SSH connection to the host specified and run the
command specified by the config parameter ```lfs.sshservercmd```, which if not
specified defaults to ```git-lfs-ssh-serve```. Simply copying this program onto your
server (no dependencies required, it's stand-alone and works on Windows, Linux
and Mac servers) and providing authenticated SSH users access to it is enough to
provide a reference implementation of a pure SSH LFS server on your own host.

## Installation ##

Just build the binary git-lfs-ssh-serve using 'go build' or cross-compile using
gox (https://github.com/mitchellh/gox). Install this on your server, on the path
of any SSH user you need to have access.

## sshd configuration for groups ##

On many Linux distros, 'ssh url command' uses a default umask of 022 which means
that uploaded file permissions are read only except for the user. If you want 
people to use their own username in their SSH url & give permission to files via
groups, you should edit /etc/pam.d/sshd and add:
```
# Setting UMASK for all ssh based connections (ssh, sftp, scp)
# always allow group perms
session    optional     pam_umask.so umask=0002
```

git-lfs-ssh-serve will copy the permissions of the base path when creating new files
& directories but it can't do that if the umask filters out the write bits. You
can't fix this with 'umask' in /etc/profile because that only applies to
interactive ssh terminals, not 'ssh url command' forms.

## Invocation ##

git-lfs will generally handle this, but to invoke the server binary you simply
need to run it by name and pass a single 'path' argument. This path is to
support multiple binary stores on the remote server end; you might want to have
a separate binary store for each repo, or for each user, or for each team, or
just a single path for everything (binaries are immutable so technically can be
shared between everyone, if permissions aren't an issue).

When given an SSH URL for the remote store, git-lfs will simply strip off the
path element and pass that as an argument to git-lfs-ssh-serve over the SSH
connection. It's up to you to use an SSH URL that reflects how you want to
partition up the remote binary store(s).

Examples:

| URL | Server command |
|-----|----------------|
|ssh://steve@bighost.com/goteam/repo1|```git-lfs-ssh-serve goteam/repo1```|
|git@thehost.com:projects/newproject|```git-lfs-ssh-serve projects/newproject```|
|ssh://andy@bighost.com//var/shared/rooted/repo|```git-lfs-ssh-serve /var/shared/rooted/repo``` (disallowed by default config)|

Rooted paths are disallowed by the default configuration for security, forcing
all repositories to be under a base path (see below).

## Configuration files ##

Configuration is via a simple key-value text file placed in the following locations:

Windows:

* %USERPROFILE%\git-lfs-serve.ini
* %PROGRAMDATA%\git-lfs\git-lfs-serve.ini

Linux/Mac:

* ~/.git-lfs-serve
* /etc/git-lfs-serve.conf

Usually you'll want to use a global config file to avoid each user having to
configure it themselves, unless you use a generic user name for all connections
and want to keep the settings there instead of system-wide.

## Configuration settings ##

There are no grouping levels in the configuration file, it's just a simple name 
= value style.

| Setting | Description | Default |
|---------|-------------|---------|
|base-path|The base directory of the binary store. Paths passed as arguments will be evaluated relative to this directory, unless they're intentionally rooted (disallowed by default, see allow-absolute) |None|
|allow-absolute-paths|Whether to allow absolute paths as arguments, i.e. rooted paths which go outside base-path. Not advisable to enable since can be a security risk.|False|
|enable-delta-receive|Whether to support receiving binary deltas to save upload time at the expense of some CPU/Memory usage to apply them. Applying patches is not as costly as generating them which is why there are separate settings|True|
|enable-delta-send|Whether to support generating deltas between binaries for clients to download. Generating deltas can be costly so you may want to disable this if you're finding it too much of an overhead.|True|
|delta-cache-path|Where to store cached deltas between versions, to avoid having to recalculate them all the time|$base-path/.deltacache|
|delta-size-limit|The maximum size file that we will attempt to use as a base for calculating a binary delta. Large files can use a lot of memory to calculate deltas on, so this limits what we attempt to use as a base. We still calculate deltas above this size but only the first X bytes are used as a base, meaning the diff can be a little less optimal at the expense of a known max memory overhead. |2147483648 (2GB)|




