import { login } from "masto";

// Visibility of the posted status. Enumerable oneOf public, unlisted, private, direct.
const VISIBILITY_OPTIONS = {
  DIRECT: "direct",
  PUBLIC: "public",
  UNLISTED: "unlisted",
  FOLLOWERS_ONLY: "followers_only",
};

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
    const message = core.getInput("message", { required: true });
    let visibility = core.getInput("visibility");

    // throw an error when the message is undefined
    if (!message) {
      throw new Error("Need to provide content to be published");
    }

    // in case visibility is undefined, we will set "public" as a default value
    if (!visibility) {
      visibility = VISIBILITY_OPTIONS.PUBLIC;
    }

    // check whether visibility has an allowed value
    const allowedVisibilities = [
      VISIBILITY_OPTIONS.DIRECT,
      VISIBILITY_OPTIONS.PUBLIC,
      VISIBILITY_OPTIONS.UNLISTED,
      VISIBILITY_OPTIONS.FOLLOWERS_ONLY,
    ];
    if (!allowedVisibilities.includes(visibility)) {
      throw new Error(
        "Visibility must be one of the following values: direct, public, unlisted, followers-only"
      );
    }

    // send the message
    // todo(cbrgm): Should we trim the message content if it exceeds the 500 character limit, or throw an error?
    let result;
    const masto = await login({
      url: mastodonURL,
      accessToken: mastodonAccessToken,
      timeout: 3 * 60 * 10,

      // should be enabled for backward-compatibility
      // see: https://github.com/neet/masto.js/pull/667
      disableVersionCheck: true,
    });

    result = await masto.statuses.create({
      status: message,
      visibility: visibility,
    });

    // outputs
    const time = new Date().toTimeString();
    console.log("Toot posted!");
    core.setOutput("ts", time);
    core.setOutput("url", result.url);
  } catch (err) {
    core.setFailed(err);
  }
}

const core = require("@actions/core");
mastodonSend(core);
