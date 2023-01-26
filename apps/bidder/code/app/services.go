package app

import (
	"bidder/code/metrics"
	"fmt"
	"time"

	"bidder/code/cache"
	"bidder/code/database/aerospike"
	"bidder/code/database/api"
	"bidder/code/database/dynamodb"
	dynamodbAudience "bidder/code/database/dynamodb/repositories/audience"
	dynamodbBudget "bidder/code/database/dynamodb/repositories/budget"
	dynamodbCampaign "bidder/code/database/dynamodb/repositories/campaign"
	dynamodbDevice "bidder/code/database/dynamodb/repositories/device"
	"bidder/code/stream"

	"emperror.dev/errors"
	as "github.com/aerospike/aerospike-client-go"
	asl "github.com/aerospike/aerospike-client-go/logger"
	"github.com/rs/zerolog/log"
)

// services contains all services required by the app to serve.
type services struct {
	stream *stream.Stream
	cache  *cache.Cache
	repo   *repository
}

// repository contains all database related services.
type repository struct {
	audience       api.AudienceRepository
	budget         api.BudgetRepository
	campaign       api.CampaignRepository
	device         api.DeviceRepository
	closeAerospike func()
}

func (r repository) Close() {
	if r.closeAerospike != nil {
		r.closeAerospike()
	}
}

// initializeServices initializes and starts services.
func initializeServices(cfg Config) (*services, error) {
	metrics.Start()

	repositories := (*repository)(nil)
	err := error(nil)

	switch cfg.DatabaseClient {
	case clientDynamodb:
		repositories, err = initializeDynamodb(cfg)
		if err != nil {
			return nil, errors.Wrap(err, "error during DynamoDB repositories initialization")
		}
	case clientAerospike:
		repositories, err = initializeAerospike(cfg)
		if err != nil {
			return nil, errors.Wrap(err, "error during Aerospike repositories initialization")
		}
	default:
		return nil, fmt.Errorf(
			"invalid database client specified: '%s'. Only '%s' and '%s' values are allowed",
			cfg.DatabaseClient,
			clientDynamodb,
			clientAerospike,
		)
	}

	newCache, err := initializeCache(cfg.Cache, repositories)
	if err != nil {
		return nil, errors.Wrap(err, "error during cache initialization")
	}

	newStream, err := initializeStream(cfg.Stream)
	if err != nil {
		return nil, errors.Wrap(err, "error during kinesis initialization")
	}

	return &services{
		stream: newStream,
		cache:  newCache,
		repo:   repositories,
	}, nil
}

// close closes services.
func (s *services) close() {
	s.stream.Close()
	s.cache.Stop()
	s.repo.Close()

	metrics.Close()
}

// initializeStream creates new stream service and waits until all streams are ready.
func initializeStream(cfg stream.Config) (*stream.Stream, error) {
	dataStream, err := stream.NewStream(cfg)
	if err != nil {
		return nil, err
	}

	log.Info().Msg("waiting for kinesis streams...")
	start := time.Now()

	if err := dataStream.WaitUntilStreamsExist(); err != nil {
		return nil, err
	}

	log.Info().Msgf("kinesis stream initialized, waited %v", time.Since(start))
	return dataStream, nil
}

// initializeDynamodb creates new repository service and checks if all tables are accessible.
func initializeDynamodb(cfg Config) (*repository, error) {
	log.Info().Msg("connecting to DynamoDB...")
	start := time.Now()

	db, err := dynamodb.NewClient(cfg.Dynamodb)
	if err != nil {
		return nil, err
	}

	audienceRepo, err := dynamodbAudience.NewRepository(cfg.Dynamodb.AudienceRepository, db)
	if err != nil {
		return nil, err
	}

	budgetRepo, err := dynamodbBudget.NewRepository(cfg.Dynamodb.BudgetRepository, db)
	if err != nil {
		return nil, err
	}

	campaignRepo, err := dynamodbCampaign.NewRepository(cfg.Dynamodb.CampaignRepository, db)
	if err != nil {
		return nil, err
	}

	deviceRepo, err := dynamodbDevice.NewRepository(cfg.Dynamodb.DeviceRepository, db)
	if err != nil {
		return nil, err
	}

	log.Info().Msgf("connected to databases, waited %v", time.Since(start))
	return &repository{
		audience: audienceRepo,
		budget:   budgetRepo,
		campaign: campaignRepo,
		device:   deviceRepo,
	}, nil
}

// initializeAerospike connects to to Aerospike database and returns *repositories
func initializeAerospike(cfg Config) (*repository, error) {
	log.Info().Msg("connecting to Aerospike...")
	startTime := time.Now()

	switch cfg.Aerospike.ClientLogLevel {
	case "INFO":
		asl.Logger.SetLevel(asl.INFO)
	case "DEBUG":
		asl.Logger.SetLevel(asl.DEBUG)
	case "ERR":
		asl.Logger.SetLevel(asl.ERR)
	default:
		asl.Logger.SetLevel(asl.OFF)
	}

	client, err := as.NewClient(cfg.Aerospike.Host, cfg.Aerospike.Port)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to Aerospike")
	}

	if cfg.Aerospike.WarmUpCount != 0 {
		_, err = client.WarmUp(cfg.Aerospike.WarmUpCount)
		if err != nil {
			client.Close()
			return nil, errors.Wrap(err, "error while warming up connection pool")
		}
	}

	aerospikeClient := aerospike.NewAerospike(client, cfg.Aerospike)

	log.Info().Dur("connecting_duration", time.Since(startTime)).Msgf("connected to Aerospike")
	log.Info().Msgf("discovered Aerospike nodes: %v", client.GetNodes())

	audienceRepo, err := aerospike.NewAudienceRepository(aerospikeClient)
	if err != nil {
		aerospikeClient.Close()
		return nil, errors.Wrap(err, "error during creating Aerospike audience repository")
	}

	budgetRepo, err := aerospike.NewBudgetRepository(aerospikeClient, cfg.Aerospike)
	if err != nil {
		aerospikeClient.Close()
		return nil, errors.Wrap(err, "error during creating Aerospike budget repository")
	}

	campaignRepo, err := aerospike.NewCampaignRepository(aerospikeClient)
	if err != nil {
		aerospikeClient.Close()
		return nil, errors.Wrap(err, "error during creating Aerospike campaign repository")
	}

	deviceRepo, err := aerospike.NewDeviceRepository(aerospikeClient)
	if err != nil {
		aerospikeClient.Close()
		return nil, errors.Wrap(err, "error during creating Aerospike device repository")
	}

	repositories := &repository{
		audience:       audienceRepo,
		budget:         budgetRepo,
		campaign:       campaignRepo,
		device:         deviceRepo,
		closeAerospike: aerospikeClient.Close,
	}

	return repositories, nil
}

// initializeCache creates new cache service and downloads all required repository tables.
func initializeCache(cfg cache.Config, db *repository) (*cache.Cache, error) {
	log.Info().Msg("initializing caches...")
	start := time.Now()

	newCache, err := cache.New(
		cfg,
		db.audience,
		db.budget,
		db.campaign,
		db.device,
	)
	if err != nil {
		return nil, err
	}

	log.Info().Msgf("caches initialized, waited %v", time.Since(start))
	return newCache, nil
}
