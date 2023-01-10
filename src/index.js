import { login } from "masto";

// Visibility of the posted status. Enumerable oneOf public, unlisted, private, direct.
const VISIBILITY_LEVEL = {
  DIRECT: "direct",
  PUBLIC: "public",
  UNLISTED: "unlisted",
  FOLLOWERS_ONLY: "private",
};

const MAX_CHARS_COUNT = 500;

async function mastodonSend(core) {
  try {
    // environment variables
    const mastodonURL = process.env.MASTODON_URL;
    const mastodonAccessToken = process.env.MASTODON_ACCESS_TOKEN;

    // check whether required env vars are set
    if (
      mastodonURL === undefined ||
      mastodonURL.length <= 0 ||
      mastodonAccessToken === undefined ||
      mastodonAccessToken.length <= 0
    ) {
      throw new Error("Need to provide MASTODON_URL and MASTODON_ACCESS_TOKEN");
    }

    // inputs
    let message = core.getInput("message", { required: true });
    let visibility = core.getInput("visibility");

    // throw an error when the message is undefined
    if (!message) {
      throw new Error("Need to provide content to be published");
    }

    message = trimMessage(message, MAX_CHARS_COUNT);

    // in case visibility is undefined, we will set "public" as a default value
    if (!visibility) {
      visibility = VISIBILITY_LEVEL.PUBLIC;
    }

    // check whether visibility has an allowed value
    const allowedVisibilities = [
      VISIBILITY_LEVEL.DIRECT,
      VISIBILITY_LEVEL.PUBLIC,
      VISIBILITY_LEVEL.UNLISTED,
      VISIBILITY_LEVEL.FOLLOWERS_ONLY,
    ];
    if (!allowedVisibilities.includes(visibility)) {
      throw new Error(
        "Visibility must be one of the following values: direct, public, unlisted, followers-only"
      );
    }

    // send the message
    let result;
    const masto = await login({
      url: mastodonURL,
      accessToken: mastodonAccessToken,
      timeout: 3 * 60 * 10,

      // should be enabled for backward-compatibility
      // see: https://github.com/neet/masto.js/pull/667
      disableVersionCheck: true,
    });

    result = await masto.v1.statuses.create({
      status: message,
      visibility: visibility,
    });

    // outputs
    const time = new Date().toTimeString();
    console.log("Toot successfully published!");
    core.setOutput("ts", time);
    core.setOutput("url", result.url);
  } catch (err) {
    core.setFailed(err);
  }
}

function trimMessage(message, n) {
  return message.length > n ? message.slice(0, n - 1) + "&hellip;" : message;
}

const core = require("@actions/core");
mastodonSend(core);
