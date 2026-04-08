package audio

type channelStream[S Sample] struct {
	Stream[S]
	channel int
}

func SelectChannel[S Sample](stream Stream[S], channel int) Stream[S] {
	return channelStream[S]{
		Stream:  stream,
		channel: channel,
	}
}

func (c channelStream[S]) Read(samples []S) (int, error) {
	config := c.Config()

	n, err := c.Stream.Read(samples)

	if n > 0 {
		for idx := range samples[:n] {
			// clear all non selected channels
			if idx%config.Channels != c.channel {
				samples[idx] = 0
			}
		}
	}

	return n, err
}
