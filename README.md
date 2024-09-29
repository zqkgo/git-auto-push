Automatically **pull -> commit -> push** to github repositories.

## 🔨 How to use

1. `git clone https://github.com/zqkgo/git-auto-push`
2. customize `config.json`
3. run `./run.sh`
4. check log periodically `tail -f git-auto-push.log`

Or you can download [the released binaries](https://github.com/zqkgo/git-auto-push/releases) directly.

```
mkdir git-auto-push
cd git-auto-push
wget -O git-auto-push https://github.com/zqkgo/git-auto-push/releases/download/v0.0.1/macos_git-auto-push
chmod +x git-auto-push
touch config.json
./git-auto-push
```

Remember customize your config file and happy syncing 🤘