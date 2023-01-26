package blackbox

import (
	bidFixtures "bidder/tests/fixtures/bid"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// Test if bidder as a whole is capable of returning bidresponse and writing to Kinesis.
func (s *blackboxSuite) TestBidder() {
	client := &http.Client{}

	req, err := http.NewRequest(
		"POST",
		s.cfg.BidderHost+s.cfg.App.Server.Address+s.cfg.App.Server.BidRequestPath,
		strings.NewReader(bidFixtures.BidRequest3),
	)
	s.Require().NoError(err)

	resp, err := client.Do(req)
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp.StatusCode)

	_, err = ioutil.ReadAll(resp.Body)
	s.Require().NoError(err)
	s.Require().NoError(resp.Body.Close())

	// Make the request for the second time, to make sure
	// first request didn't introduce corrupt state.
	resp2, err := client.Do(req)
	s.Require().NoError(err)
	s.Equal(http.StatusOK, resp2.StatusCode)

	_, err = ioutil.ReadAll(resp2.Body)
	s.Require().NoError(err)
	s.Require().NoError(resp2.Body.Close())

	// Wait for Kinesis producers to flush.
	s.Require().Eventually(func() bool {
		return len(s.getStreamRecords(s.cfg.App.Stream.Producer.StreamName)) > 0
	}, time.Second*15, time.Second)

	bidRequests := s.getStreamRecords(s.cfg.App.Stream.Producer.StreamName)
	s.Require().Len(bidRequests, 1)
}

// Test if bidder returns status 204 in case of an unknown device ID.
func (s *blackboxSuite) TestBidderNoBid() {
	client := &http.Client{}

	req, err := http.NewRequest(
		"POST",
		s.cfg.BidderHost+s.cfg.App.Server.Address+s.cfg.App.Server.BidRequestPath,
		strings.NewReader(bidFixtures.NoBidBidRequest3),
	)
	s.Require().NoError(err)

	resp, err := client.Do(req)
	s.Require().NoError(err)
	s.Equal(http.StatusNoContent, resp.StatusCode)

	s.Require().NoError(resp.Body.Close())
}
