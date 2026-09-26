package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
	"tramflow/internal/config"
	"tramflow/internal/contract"
	d "tramflow/internal/domain"
	"tramflow/internal/inference"
	"tramflow/internal/storage"
)

func run() error {
	bundle := flag.String("bundle", "", "Prepared ML/ETL bundle JSON")
	migrate := flag.Bool("migrate", false, "Apply database migrations")
	synthetic := flag.Bool("allow-synthetic", false, "Allow explicitly synthetic fixture publication")
	gc := flag.Bool("gc-data", false, "Collect unreferenced database profiles; keeps manifests/artifacts and tombstones")
	expire := flag.Bool("expire", false, "Expire retained inactive snapshots")
	createUser := flag.String("create-user", "", "Provision user using API_PASSWORD environment")
	flag.Parse()
	c, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	s, err := storage.Open(ctx, c.Database, c.Pool)
	if err != nil {
		return err
	}
	defer s.Pool.Close()
	if *migrate {
		if err = s.Migrate(ctx); err != nil {
			return err
		}
	}
	if *createUser != "" {
		if err = s.ProvisionUser(ctx, *createUser, os.Getenv("API_PASSWORD")); err != nil {
			return err
		}
	}
	if *expire {
		if err = s.Expire(ctx); err != nil {
			return err
		}
		if !*gc {
			return nil
		}
	}
	if *gc {
		return s.CollectData(ctx)
	}
	if *bundle == "" {
		if *migrate || *createUser != "" {
			return nil
		}
		return fmt.Errorf("--bundle is required")
	}
	f, err := os.Open(*bundle)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, 32*1024*1024+1))
	if err != nil {
		return err
	}
	if len(b) > 32*1024*1024 {
		return fmt.Errorf("bundle exceeds 32 MiB")
	}
	if err = contract.UniqueJSON(b); err != nil {
		return err
	}
	var input d.Bundle
	dec := json.NewDecoder(bytesReader(b))
	dec.DisallowUnknownFields()
	if err = dec.Decode(&input); err != nil {
		return err
	}
	spec, err := contract.Load(c.Contract)
	if err != nil {
		return err
	}
	models := inference.New(c.Artifacts)
	defer models.Close()
	if err = storage.ValidateBundle(ctx, &input, spec, models, *synthetic); err != nil {
		return err
	}
	if err = s.Publish(ctx, input); err != nil {
		return err
	}
	fmt.Println("Published snapshot:", input.Snapshot.Provenance.SnapshotID)
	return nil
}
func bytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
