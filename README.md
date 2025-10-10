# Roast

A Go backend for my API service.

And powered by Gin.

## Feature

- Atom IT Club's some basic backend functions.  (Thanks to [@Chemio9](https://github.com/chemio9) for his great contribution to our club. Without your inspiration and help, I wouldn't be able to complete this work.)
- Redirect Service
- Email Sender (Support SMTP and Microsoft Exchange)
- Avatar provider

## Usage

### Start using

Because of the source code is imperfect, I will update it frequently.

Firstly, clone it into your device.

```bash
git clone https://github.com/Koch2333/Backend-Go.git
```

Before running, please run the following command to scan and mount the module.
```bash
go generate ./internal/bootstrap/mod
```

And, run it directly, the Go module will be download automatically.

```bash
go build backend-go
```

## To do
- Divide the function module to specific repositories.
- Connect backend and frontend.
- API Documents.
