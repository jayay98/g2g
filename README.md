# g2g
A set of command-utilities for hosting git-based code repositories. Ideally, you can host your git server in your home computer, and the server is accessible from every point in the internet, without uploading the codes to a centralized archive. The helpers are based on libp2p and no additional router configurations are needed.

## Usage
On the server-side:
```sh
>> git g2g
Host ID: QmafQq3BfH1b1hF6p8tcvnn5opxmPuPQQtuSesRzSgBvKY
```

And then on the client side:
```sh
>> git clone g2g://QmafQq3BfH1b1hF6p8tcvnn5opxmPuPQQtuSesRzSgBvKY/<repository>.git
```