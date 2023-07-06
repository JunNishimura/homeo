include .env

.PHONY: build deploy

build:
	sam build

deploy: build
	sam deploy --parameter-overrides "LineChannelSecret=$(LINE_CHANNEL_SECRET) LineChannelToken=$(LINE_CHANNEL_TOKEN) OpenAIKey=$(OPEN_AI_KEY)" --profile homeo --guided
