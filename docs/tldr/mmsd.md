# mmsd

> MET Messaging System Daemon
> More information: <https://github.com/metno/go-mms>.
> Part of MMS: <https://github.com/metno/mms>

- Load the MMS module
`module load mms`

- Start the daemon for testing:
`mmsd`

- Generate an API key:
`mmsd keys --gen -m "test key"`

- Generate a certificate signing request:

`mmsd gencsr`

- Start the daemon for production:
`mmsd -tls`
