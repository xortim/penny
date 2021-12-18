# Penny - the community moderation Slack Bot

Penny is a chat bot designed to assist with community moderation. This chatbot
uses a lightweight Slack bot framework called
[Gadget](https://github.com/gadget-bot/gadget/).

So why the name "Penny"? Well, that's easy - if you've ever seen the cartoon
Inspector Gadget, Penny is Gadget's niece who happened to do all the work.

# Building

Run `make help` to see all targets. Use `make build` to get a binary or `make
container` to get a container image.

# Setup

Setting up a new Slack bot is unique. Using the supplied manifest file in this
repo simplifies the process.

## Create the App in Slack

First, create your Slack org. Once created you'll need to access the Workspace
Settings `https://${ORG_NAME}.slack.com/admin/settings`

Once in the settings, follow these steps:

1. Go to "Configure Apps"
1. Click on "Build" in the top right.
1. Click "Create New App"
1. Select the "From an App Manifest" option.
1. Update the `settings.event_subscriptions.request_url` value to reflect where
   you'll be hosting Penny.

Not done yet. You'll need a couple of values that Slack generated for your bot.
There's a table in in the Configuration section for how to set these.

### Classic

On your App's settings navigate to "Basic Information" and grab the Signing
secret.

Under the "OAuth & Permissions" page, grab both the "User OAuth Token" and "Bot
User OAuth Token"

### New

Slack is changing how Apps are configured. This will get updated later.

## Configuration

Penny uses Viper for configuration. Gadget does not. This will be easier later,
but for now you need to configure two things. Gadget, and Penny.

[Penny honors](https://github.com/xortim/penny/blob/develop/cmd/root.go) the
Gadget environment variables for the following keys. This is to support passing
configuring into Gadget from Penny at a later date. These values **must** be set
as environment variables.

| Value                | Gadget ENV             | Penny Key                    |
| -------------------- | ---------------------- | ---------------------------- |
| Database Host        | `GADGET_DB_HOST`       | `db.host`                    |
| Database Name        | `GADGET_DB_NAME`       | `db_name`                    |
| Database Password    | `GADGET_DB_PASS`       | `db_pass`                    |
| Database Username    | `GADGET_DB_USER`       | `db_user`                    |
| Global Admins        | `GADGET_GLOBAL_ADMINS` | `slack.global_admins`        |
| Server Port          | `GADGET_SERVER_PORT`   | `server.port`                |
| Signing Secret       | `SLACK_SIGNING_SECRET` | `slack.signing_secret`       |
| User OAuth Token     | `SLACK_OAUTH_TOKEN`    | `slack.user_oauth_token`     |


The following values must be set as Penny configuration keys:

| Value                | Gadget ENV             | Penny Key                    |
| -------------------- | ---------------------- | ---------------------------- |
| Bot User OAuth Token | n/a                    | `slack.bot_oauth_token`      |


If this is all confusing. That's because it is. I don't know what I'm doing.


Example `.env` file for Gadget:

```
export SLACK_OAUTH_TOKEN=xoxb-XXXXXXXXXXX-XXXXXXXXXXXXX-XXXXXXXXXXXXXXXXXXXXXXXX
export SLACK_SIGNING_SECRET=XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX

export GADGET_GLOBAL_ADMINS=U0Z6G0BTM
export GADGET_SERVER_PORT=3000

# The mariaDB container want's no GADGET_ prefix. Gadget Does. Sorry :(
export DB_USER=penny
export DB_NAME=penny_dev
export DB_PASS=XXXXXXXX
export DB_ROOT_PASS=XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
export GADGET_DB_HOST=localhost
export GADGET_DB_USER=${DB_USER}
export GADGET_DB_NAME=${DB_NAME}
export GADGET_DB_PASS=${DB_PASS}
```

**NOTE**: Once compiled, simply run `penny --help` for guidance on the
configuration.

The easiest way to configure Penny is with a yaml file. Place this in
`${HOME}/.penny.yaml` and modify it to suit your needs.

```yaml
---
slack:
  bot_oauth_token: xoxb-XXXXXXXXXXX-XXXXXXXXXXXXX-XXXXXXXXXXXXXXXXXXXXXXXX

# monitor messages marked as spam.
# This requires the use of the Reacji-Channel App
# reacji-channeler.builtbyslack.com
# Specify the emoji used for community moderation of spam
# and the channel to which Reacji re-posts the message
spam_feed:
  channel: spam-feed #make sure penny is a member of this channel
  reaction_emoji_miss: shrug
  reaction_emoji_hit: no_good
  reacji_response: "I'll look into it."
  op_response: "This message has been flagged by our community as SPAM. The admins have been notified."
  activity_low_watermark: 10
  local_timezone: "America/New_York"
  max_anomaly_score: 2
  anomaly_scores:
    low_activity: 1
    reported: 2
    outside_tz: 2
```

Clear as mud? Yup. Isn't learning a new thing fun?

### Dependencies

Penny's primary feature (SPAM removal) relies on another App to be installed in
your Slack org and configured to post messages to another channel. Originally
Penny was going to include this functionality, but it's not an event that Slack
allows users to subscribe to. Otherwise we'd have to complicate Penny with more
statefulness and duplicate all the nice logic this app provides in addition to
working around the limitations of their public API (like persistently joining
public channels to monitor all reactions.)

Install the [Reacji-Channel App](reacji-channeler.builtbyslack.com) and
configure it to re-post messages with your desired reaction to the
`spam_feed.channel` channel.

**NOTE:** You will have to invite Penny to this channel.

# Running Penny

Running penny is pretty simple. There are two approaches, one for local
development and another for production.

## Local Dev

I recommend running the `make start-db` task and configuring Penny to talk to
that container. Read the `Makefile` for more information. Once you do that, run
`make build` (or use your IDE to launch Penny for debugging). Personally, I use
`ngrok` and update the Bot's configuration in my test organization accordingly.
This gets you request inspection and logs.

## Production

Use the supplied example `docker-compose.yaml` file for inspiration.

