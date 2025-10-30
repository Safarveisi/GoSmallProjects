# Automatically create a local and remote git repo

## Usage

```bash
# You can set -create-remote to false to skip creating a remote repo on GitHub
go run . -repo-name=test -repo-user=username -repo-email=firstname.lastname@gmail.com -create-remote=true
```

> [!NOTE]
> You need a GITHUB personal access token if you set -create-home to true. Your personal
> token should have Administration permissions (read and write).