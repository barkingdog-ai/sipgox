# sipgox

is experimental/extra area to add more functionality on top [sipgo lib](https://github.com/emiago/sipgo),

To find out more check [GO Documentation](https://pkg.go.dev/github.com/emiago/sipgox)

To find out more, read also article about [E2E testing](https://github.com/emiago/sipgo/wiki/E2E-testing) and check [GO Documentation](https://pkg.go.dev/github.com/emiago/sipgox)

---

If you find it useful, support [sipgo lib](https://github.com/emiago/sipgo), open issue etc...

Checkout [echome](/echome/) example to see more. 

## Phone (Deprecated, use diago)

Features:
- [x] Simple API for UA/phone build with dial answer register actions
- [x] Minimal SDP package for audio
- [x] RTP/RTCP receiving and logging
- [x] Extendable MediaSession handling for RTP/RTCP handling (ex microphone,speaker)
- [x] Hangup control on caller
- [x] Timeouts handling
- [x] Digest auth
- [x] Transfers on phone answer, dial

Phone is wrapper that allows you to build phone in couple of lines. 
Then you can quickly create/receive SIP call, handle RTP/RTCP, etc... 
It uses `sipgo.Dialog` and `media` package.

*NOTE*: It has specific design for testing, and it can not be used for full softphone build.

### Dialer

```go
ua, _ := sipgo.NewUA()
defer ua.Close()

// Create a phone
phone := sipgox.NewPhone(ua) 

// Run dial
ctx, _ := context.WithTimeout(context.Background(), 60*time.Second)

// Blocks until call is answered
dialog, err := phone.Dial(ctx, sip.Uri{User:"bob", Host: "localhost", Port:5060}, sipgox.DialOptions{})
if err != nil {
    // handle error
    return
}
defer dialog.Close() // Close dialog for cleanup

select {
case <-dialog.Done():
    return
case <-time.After(5 *time.Second):
    dialog.Hangup(context.TODO())
}
```

### Receiver

```go
ctx, _ := context.WithCancel(context.Background())

ua, _ := sipgo.NewUA()
defer ua.Close()

// Create a phone
phone := sipgox.NewPhone(ua)

// Blocks until call is answered
dialog, err := phone.Answer(ctx, sipgox.AnswerOptions{
    Ringtime:  5* time.Second,
})
if err != nil {
    //handle error
    return
}
defer dialog.Close() // Close dialog for cleanup

select {
case <-dialog.Done():
    return
case <-time.After(10 *time.Second):
    dialog.Hangup(context.TODO())
}
```

### Reading/Writing RTP/RTCP on dialog

After you Answer or Dial on phone, you receive dialog.

**RTP**
```go
buf := make([]byte, media.RTPBufSize) // Has MTU size
pkt := rtp.Packet{}
err := dialog.ReadRTP(buf, &pkt)

err := dialog.WriteRTP(pkt)

```

similar is for RTCP
