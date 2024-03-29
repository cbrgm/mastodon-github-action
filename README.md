# Mastodon Send GitHub Action

<img
  src="https://upload.wikimedia.org/wikipedia/commons/4/48/Mastodon_Logotype_%28Simple%29.svg"
  width="120px"
  align="right"
/>

**Use this action to send a toot (message) from a GitHub actions workflow to Mastodon.**

[![GitHub release](https://img.shields.io/github/release/cbrgm/mastodon-github-action.svg)](https://github.com/cbrgm/mastodon-github-action)
[![Go Report Card](https://goreportcard.com/badge/github.com/cbrgm/mastodon-github-action)](https://goreportcard.com/report/github.com/cbrgm/mastodon-github-action)
[![go-lint-test](https://github.com/cbrgm/mastodon-github-action/actions/workflows/go-lint-test.yml/badge.svg)](https://github.com/cbrgm/mastodon-github-action/actions/workflows/go-lint-test.yml)
[![go-binaries](https://github.com/cbrgm/mastodon-github-action/actions/workflows/go-binaries.yml/badge.svg)](https://github.com/cbrgm/mastodon-github-action/actions/workflows/go-binaries.yml)
[![container](https://github.com/cbrgm/mastodon-github-action/actions/workflows/container.yml/badge.svg)](https://github.com/cbrgm/mastodon-github-action/actions/workflows/container.yml)

## Inputs

- `url`: **Required** - Mastodon instance URL.
- `access-token`: **Required** - Mastodon access token for authentication. Use secrets to protect your access token.
- `message`: **Required** - The content of the toot to be posted.
- `visibility`: Optional - Visibility of the toot (`public`, `unlisted`, `private`, `direct`). Defaults to `public`.
- `sensitive`: Optional - Mark the toot and attached media as sensitive. Accepts `true` or `false`. Defaults to `false`.
- `spoiler-text`: Optional - Text to be shown as a warning before the actual content, used when `sensitive` is `true`.
- `language`: Optional - ISO 639 language code for the toot, helping to categorize the post by language.
- `scheduled-at`: Optional - ISO 8601 Datetime when the toot should be posted. Must be at least 5 minutes in the future.

## Outputs

- `id`: The ID of the toot that was posted. Useful for further actions or logging.
- `url`: The URL to the toot if present. Allows direct access to the posted toot.
- `scheduled_at`: The datetime at which the toot is scheduled to be posted, in ISO 8601 format if the toot was scheduled for a future time.

## Container Usage

This action can be executed independently from workflows within a container. To do so, use the following command:

```
podman run --rm -it ghcr.io/cbrgm/mastodon-github-action:v2 --help
```

## Workflow Usage

First, open `/settings/applications/new` of your instance on your browser and create new application. Once the application is created set the following repository secrets

* `MASTODON_URL` - Your instance URL, e.g. `https://example.social`
* `MASTODON_ACCESS_TOKEN` - Your access token obtained from your newly created application

Use the following step in your GitHub Actions Workflow:

```yaml

- name: Send toot to Mastodon
  id: mastodon
  uses: cbrgm/mastodon-github-action@v2
  with:
    access-token: ${{ secrets.MASTODON_ACCESS_TOKEN }} # access token
    url: ${{ secrets.MASTODON_URL }} # https://example.social
    message: "Hello from GitHub Actions!"

```

Advanced usage:

```yaml
- name: Send toot to Mastodon with additional options
  id: mastodon_toot
  uses: cbrgm/mastodon-github-action@v2
  with:
    access-token: ${{ secrets.MASTODON_ACCESS_TOKEN }} # Mastodon access token for authentication.
    url: ${{ secrets.MASTODON_URL }} # Mastodon instance URL, e.g., https://example.social.
    message: "Hello from GitHub Actions! Check out our latest update." # The content of the toot.
    visibility: "unlisted" # Make the toot unlisted to avoid spamming public timelines.
    sensitive: "true" # Mark the toot as sensitive.
    spoiler-text: "Latest Update" # Provide a content warning for the actual message.
    language: "en" # ISO 639 language code for the toot.
    scheduled-at: "2024-01-01T00:00:00Z" # Schedule the toot for a future date/time.

# Example on how to use outputs from the Mastodon action step.
- name: Get toot information
  run: |
    echo "Toot ID: ${{ steps.mastodon_toot.outputs.id }}"
    echo "Toot URL: ${{ steps.mastodon_toot.outputs.url }}"
    echo "Scheduled at: ${{ steps.mastodon_toot.outputs.scheduled_at }}"
```

You can find more usage examples in the [./example-workflows](./example-workflows/) subfolder.

#### About message `visibility` types

- **Public**: Posts are visible to everyone, including those outside the Fediverse. They can be found in Mastodon searches and on a user's public profile. Represented by a globe icon 🌎.
- **Unlisted**: Posts are visible to everyone but do not appear in trending lists, Local or Federated timelines, or search results. Useful for replies in threads to avoid cluttering timelines. Marked with an open lock icon.
- **Followers-only**: Only the poster's followers can see these posts. Advisable to enable follower requests to control who sees these posts. Indicated by a lock 🔒 or people 👥 icon.
- **Mentioned**: Posts are only visible to users mentioned in the post. Use cautiously to ensure privacy. Denoted by an @ symbol.

## Contributing & License

* **Contributions Welcome!**: Interested in improving or adding features? Check our [Contributing Guide](https://github.com/cbrgm/mastodon-github-action/blob/main/CONTRIBUTING.md) for instructions on submitting changes and setting up development environment.
* **Open-Source & Free**: Developed in my spare time, available for free under [Apache 2.0 License](https://github.com/cbrgm/mastodon-github-action/blob/main/LICENSE). License details your rights and obligations.
* **Your Involvement Matters**: Code contributions, suggestions, feedback crucial for improvement and success. Let's maintain it as a useful resource for all 🌍.
