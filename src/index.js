import { login } from "masto";

const VISIBILITY_OPTIONS = {
	DIRECT: "direct",
	PUBLIC: "public",
	UNLISTED: "unlisted",
	FOLLOWERS_ONLY: "followers_only",
};

async function mastodonSend(core) {
	try {
		const mastodonURL = process.env.MASTODON_URL;
		const mastodonAccessToken = process.env.MASTODON_ACCESS_TOKEN;
		if (
			mastodonURL === undefined ||
			mastodonURL.length <= 0 ||
			mastodonAccessToken === undefined ||
			mastodonAccessToken.length <= 0
		) {
			throw new Error("Need to provide MASTODON_URL and MASTODON_ACCESS_TOKEN");
		}
		const message = core.getInput("message", { required: true });
		let visibility = core.getInput("visibility");
		if (!message) {
			throw new Error("Need to provide content to be published");
		}

		if (!visibility) {
			visibility = VISIBILITY_OPTIONS.PUBLIC;
		}
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
		let result;

		const masto = await login({
			url: mastodonURL,
			accessToken: mastodonAccessToken,
			timeout: 3 * 60 * 10,
			disableVersionCheck: true,
		});

		result = await masto.statuses.create({
			status: message,
			visibility: visibility,
		});

		const time = new Date().toTimeString();
		console.log("Toot posted!");
		core.setOutput("ts", time);
		core.setOutput("url", result.url);

	} catch (err) {
		core.setFailed(err);
	}
}

const core = require('@actions/core');
mastodonSend(core);
