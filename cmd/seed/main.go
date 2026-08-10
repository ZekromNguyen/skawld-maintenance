package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	assetpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/asset/adapter/postgres"
	assetapp "github.com/ZekromNguyen/skawld-maintenance/internal/asset/application"
	attachmentpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/attachment/adapter/postgres"
	attachmentapp "github.com/ZekromNguyen/skawld-maintenance/internal/attachment/application"
	copilotpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/copilot/adapter/postgres"
	copilotapp "github.com/ZekromNguyen/skawld-maintenance/internal/copilot/application"
	demonstrationapp "github.com/ZekromNguyen/skawld-maintenance/internal/demonstration/application"
	executionpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/execution/adapter/postgres"
	executionapp "github.com/ZekromNguyen/skawld-maintenance/internal/execution/application"
	handoverpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/handover/adapter/postgres"
	handoverapp "github.com/ZekromNguyen/skawld-maintenance/internal/handover/application"
	identitypostgres "github.com/ZekromNguyen/skawld-maintenance/internal/identity/adapter/postgres"
	identitydomain "github.com/ZekromNguyen/skawld-maintenance/internal/identity/domain"
	incidentpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/incident/adapter/postgres"
	incidentapp "github.com/ZekromNguyen/skawld-maintenance/internal/incident/application"
	knowledgepostgres "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/adapter/postgres"
	knowledgeapp "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/application"
	knowledgedomain "github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/domain"
	"github.com/ZekromNguyen/skawld-maintenance/internal/knowledge/ingest"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/audit"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/clock"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/config"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/database"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/events"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/id"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/idempotency"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/jobs"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/objectstore"
	s3store "github.com/ZekromNguyen/skawld-maintenance/internal/platform/objectstore/s3"
	reportpostgres "github.com/ZekromNguyen/skawld-maintenance/internal/report/adapter/postgres"
	reportapp "github.com/ZekromNguyen/skawld-maintenance/internal/report/application"
	"github.com/ZekromNguyen/skawld-maintenance/internal/skawld"
	workflowapp "github.com/ZekromNguyen/skawld-maintenance/internal/workflow/application"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var slogger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func main() {
	if err := run(context.Background()); err != nil {
		slogger.Error("seed failed", "error", err)
		os.Exit(1)
	}
	slogger.Info("seed complete")
}

func run(ctx context.Context) error {
	cfg, err := config.Load(config.RoleAPI)
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	pool, err := database.Open(ctx, cfg.Database, config.RoleAPI)
	if err != nil {
		return err
	}
	defer pool.Close()

	now := time.Now().UTC().Truncate(time.Second)
	c := clock.System{}
	gen := id.UUID{}
	objectStore, err := s3store.New(ctx, cfg.ObjectStore)
	if err != nil {
		return fmt.Errorf("object store: %w", err)
	}

	authorityReader := identitypostgres.AuthorityReader{Pool: pool}
	jobClient, err := jobs.NewWithWorkers(pool, cfg.Jobs, slogger, jobs.WorkerSet{
		DocumentIngestor: &documentIngestor{pool: pool, objects: objectStore, cfgDocuments: cfg.Documents},
	})
	if err != nil {
		return fmt.Errorf("job producer: %w", err)
	}
	structuredProviders, embeddingProvider, err := skawld.BuildProviders(
		skawld.AIConfig{}, nil,
	)
	if err != nil {
		return fmt.Errorf("AI provider: %w", err)
	}
	assetStore := assetpostgres.Store{Pool: pool, IDs: gen, Clock: c, Idempotency: idempotency.Store{}, Audit: audit.Sink{}}
	incidentStore := incidentpostgres.Store{Pool: pool, IDs: gen, Clock: c, Idempotency: idempotency.Store{}, Audit: audit.Sink{}}
	executionStore := executionpostgres.Store{Pool: pool, IDs: gen, Clock: c, Idempotency: idempotency.Store{}, Audit: audit.Sink{}}
	attachmentStore := attachmentpostgres.Store{Pool: pool, IDs: gen, Clock: c, Idempotency: idempotency.Store{}, Audit: audit.Sink{}, Objects: objectStore}
	knowledgeStore := knowledgepostgres.Store{Pool: pool, IDs: gen, Clock: c, Idempotency: idempotency.Store{}, Audit: audit.Sink{}, Enqueuer: jobClient, Embeddings: embeddingProvider}
	copilotStore := copilotpostgres.Store{Pool: pool, IDs: gen, Clock: c, Idempotency: idempotency.Store{}, Audit: audit.Sink{}}
	reportStore := reportpostgres.Store{Pool: pool, IDs: gen, Clock: c, Idempotency: idempotency.Store{}, Audit: audit.Sink{}}
	handoverStore := handoverpostgres.Store{Pool: pool, IDs: gen, Clock: c, Idempotency: idempotency.Store{}, Audit: audit.Sink{}}

	principal, organizationID, siteID := seedFoundation(ctx, pool, gen, c)
	seedRoleAccounts(ctx, pool, gen, organizationID, siteID, c.Now())

	// Step 1: asset + criticality (reuse existing P-302 on re-run)
	asset, ok, err := lookupAsset(ctx, pool, organizationID, siteID, "P-302")
	if err != nil {
		return fmt.Errorf("lookup asset: %w", err)
	}
	if !ok {
		asset, _, err = assetStore.Create(ctx, principal, key("asset-p302"), assetapp.CreateAsset{
			SiteID: siteID, Tag: "P-302", Name: "Process Pump P-302",
			Class: "CENTRIFUGAL_PUMP", Manufacturer: "Fictional Pump Co.",
			Model: "CFP-150", SourceOfTruth: "OWNED_BY_SKAWLD",
			Components: []assetapp.ComponentInput{
				{Code: "MTR-BRG", Name: "Motor-side bearing", Type: "BEARING"},
				{Code: "PUMP-BRG", Name: "Pump-side bearing", Type: "BEARING"},
			},
		})
		if err != nil {
			return fmt.Errorf("create asset: %w", err)
		}
	}
	if hasCriticality(ctx, pool, organizationID, asset.ID) {
		slogger.Info("criticality already approved for P-302", "asset_id", asset.ID)
	} else if _, _, err := assetStore.ApproveCriticality(ctx, principal, key("crit-p302"), asset.ID, assetapp.ApproveCriticality{
		Rating: "A", SafetyImpact: 4, ProductionImpact: 4, EnvironmentalImpact: 1,
		FinancialImpact: 3, Redundancy: "NONE", Rationale: "Single duty pump for cooling circuit",
	}); err != nil {
		return fmt.Errorf("approve criticality: %w", err)
	}

	// Step 2: incident (reuse on re-run)
	incident, ok, err := lookupIncident(ctx, pool, organizationID, "High vibration on P-302 motor bearing")
	if err != nil {
		return fmt.Errorf("lookup incident: %w", err)
	}
	if !ok {
		incident, _, err = incidentStore.Create(ctx, principal, key("incident-1"), incidentapp.CreateIncident{
			SiteID: siteID, AssetID: asset.ID, Summary: "High vibration on P-302 motor bearing",
			Priority: "HIGH", SourceOfTruth: "OWNED_BY_SKAWLD", DetectedAt: now,
		})
		if err != nil {
			return fmt.Errorf("create incident: %w", err)
		}
	}

	// Step 2b: execution (reuse an existing IN_PROGRESS/COMPLETED one on re-run)
	execution, ok, err := lookupExecution(ctx, pool, organizationID, incident.ID)
	if err != nil {
		return fmt.Errorf("lookup execution: %w", err)
	}
	if !ok {
		execution, _, err = executionStore.Create(ctx, principal, key("exec-1"), executionapp.CreateExecution{
			IncidentID: incident.ID, Purpose: "Diagnose high vibration on P-302", AssignedTo: principal.ID,
		})
		if err != nil {
			return fmt.Errorf("create execution: %w", err)
		}
	}
	executionCompleted := execution.State == "COMPLETED"
	if executionCompleted {
		slogger.Info("execution already completed; reusing it for demonstrations",
			"execution_id", execution.ID)
	}

	if !executionCompleted {
		// Step 3: run the pump inspection flow
		execution, _, err = executionStore.Start(ctx, principal, key("start-1"), execution.ID, executionapp.StartExecution{
			ExpectedVersion: execution.Version,
		})
		if err != nil {
			return fmt.Errorf("start execution: %w", err)
		}
		for _, s := range execution.Steps[:3] {
			last := execution
			result, _, err := executionStore.CompleteStep(ctx, principal, key("step-"+s.Key), execution.ID, s.ID, executionapp.CompleteStep{
				ExpectedExecutionVersion: last.Version,
				ExpectedStepVersion:      s.Version,
			})
			if err != nil {
				return fmt.Errorf("complete step %s: %w", s.Key, err)
			}
			_ = result
			execution, err = executionStore.Get(ctx, principal, execution.ID)
			if err != nil {
				return err
			}
		}
		if _, _, err := executionStore.RecordMeasurement(ctx, principal, key("meas-1"), execution.ID, executionapp.RecordMeasurement{
			ClientEventID: uuid.NewString(), ComponentID: asset.Components[0].ID,
			MeasurementType: "VIBRATION_VELOCITY", Value: "8.1", Unit: "MM_PER_S",
			Source: "MANUAL", DataQuality: "GOOD", VerificationStatus: "UNVERIFIED",
			ObservedAt: now,
		}); err != nil {
			return fmt.Errorf("record measurement: %w", err)
		}
		if _, _, err := executionStore.RecordMeasurement(ctx, principal, key("meas-2"), execution.ID, executionapp.RecordMeasurement{
			ClientEventID: uuid.NewString(), ComponentID: asset.Components[0].ID,
			MeasurementType: "TEMPERATURE", Value: "94", Unit: "DEG_C",
			Source: "INSTRUMENT", DataQuality: "GOOD", VerificationStatus: "UNVERIFIED",
			ObservedAt: now,
		}); err != nil {
			return fmt.Errorf("record temperature: %w", err)
		}
		if _, _, err := executionStore.RecordObservation(ctx, principal, key("obs-1"), execution.ID, executionapp.RecordObservation{
			ClientEventID: uuid.NewString(), ComponentID: asset.Components[0].ID,
			Status: "DEFECT", Narrative: "Bearing housing shows slight grease staining",
			Source: "TECHNICIAN", VerificationStatus: "UNVERIFIED", ObservedAt: now,
		}); err != nil {
			return fmt.Errorf("record observation: %w", err)
		}

		// Intrusive step must stay blocked until isolation verification
		intrusive := execution.Steps[3]
		blocked, _, err := executionStore.CompleteStep(ctx, principal, key("block-1"), execution.ID, intrusive.ID, executionapp.CompleteStep{
			ExpectedExecutionVersion: execution.Version,
			ExpectedStepVersion:      intrusive.Version,
		})
		if err != nil {
			return fmt.Errorf("block intrusive step: %w", err)
		}
		if blocked.State != "BLOCKED" {
			return fmt.Errorf("intrusive step state = %s, want BLOCKED", blocked.State)
		}
		if _, _, err := executionStore.VerifyPrerequisite(ctx, principal, key("prereq-1"), execution.ID, executionapp.VerifyPrerequisite{
			Type: "ENERGY_ISOLATION", Status: "VERIFIED", ExternalReference: "LOTO-P302-001",
			VerifiedAt: now,
		}); err != nil {
			return fmt.Errorf("verify prerequisite: %w", err)
		}
		execution, err = executionStore.Get(ctx, principal, execution.ID)
		if err != nil {
			return err
		}
		for _, s := range execution.Steps[3:] {
			result, _, err := executionStore.CompleteStep(ctx, principal, key("step-final-"+s.Key), execution.ID, s.ID, executionapp.CompleteStep{
				ExpectedExecutionVersion: execution.Version,
				ExpectedStepVersion:      s.Version,
			})
			if err != nil {
				return fmt.Errorf("complete intrusive step: %w", err)
			}
			_ = result
			execution, err = executionStore.Get(ctx, principal, execution.ID)
			if err != nil {
				return err
			}
		}
		if _, _, err := executionStore.RecordAction(ctx, principal, key("action-1"), execution.ID, executionapp.RecordAction{
			StepID: intrusive.ID, ComponentID: asset.Components[0].ID,
			ActionType: "LUBRICATED", Narrative: "Re-greased motor-side bearing", Outcome: "COMPLETED",
			PerformedAt: now,
		}); err != nil {
			return fmt.Errorf("record action: %w", err)
		}
		if _, _, err := executionStore.RecordDecision(ctx, principal, key("decision-1"), execution.ID, executionapp.RecordDecision{
			ClientEventID: uuid.NewString(), StepID: intrusive.ID, ComponentID: asset.Components[0].ID,
			Decision: "REPLACE_BEARING_SCHEDULED", Rationale: "Vibration above alert threshold and lubrication degraded",
			Alternatives: []string{"MONITOR"}, DecidedAt: now,
		}); err != nil {
			return fmt.Errorf("record decision: %w", err)
		}
		execution, _, err = executionStore.Complete(ctx, principal, key("complete-1"), execution.ID, executionapp.CompleteExecution{
			ExpectedVersion: execution.Version, OutcomeSummary: "Bearing re-greased; replacement scheduled",
		})
		if err != nil {
			return fmt.Errorf("complete execution: %w", err)
		}
	}

	// Step 4: knowledge document + ingestion
	doc, _, err := knowledgeStore.CreateDocument(ctx, principal, key("doc-1"), knowledgeapp.CreateDocument{
		SiteID: siteID, DocumentType: knowledgedomain.DocumentSOP,
		Title: "P-302 Bearing Lubrication SOP", Authority: knowledgedomain.AuthoritySiteApproved,
	})
	if err != nil {
		return fmt.Errorf("create document: %w", err)
	}
	revision, _, err := knowledgeStore.CreateRevision(ctx, principal, key("rev-1"), doc.ID, knowledgeapp.CreateRevision{
		Revision: "R1", Language: "en",
		Applicability: []knowledgedomain.Applicability{
			{SiteID: siteID, AssetClass: "CENTRIFUGAL_PUMP", AssetID: asset.ID},
		},
	})
	if err != nil {
		return fmt.Errorf("create revision: %w", err)
	}

	content := []byte(`P-302 bearing lubrication procedure.
Expected vibration velocity for a healthy motor-side bearing is below 7.1 mm/s.
Values above 8.0 mm/s require inspection of lubrication and alignment before considering intrusive work.
Temperature above 90 degrees Celsius indicates degraded lubrication.
Always verify energy isolation with a work permit before intrusive bearing inspection.`)
	checksum := sha256.Sum256(content)
	attachment, _, err := attachmentStore.Create(ctx, principal, key("attach-doc"), attachmentapp.CreateManifest{
		SiteID: siteID, EntityKind: "DOCUMENT_REVISION", EntityID: revision.ID,
		ClientEventID: uuid.NewString(), OriginalFilename: "p302-lubrication-sop.txt",
		DeclaredMIME: "text/plain", SizeBytes: int64(len(content)),
		ChecksumSHA256: hex.EncodeToString(checksum[:]),
	})
	if err != nil {
		return fmt.Errorf("create document attachment: %w", err)
	}
	if _, err := objectStore.Put(ctx, objectstore.PutRequest{
		Key:  fmt.Sprintf("organizations/%s/sites/%s/attachments/%s", organizationID, siteID, attachment.ID),
		Body: bytes.NewReader(content), Size: int64(len(content)), ContentType: "text/plain",
	}); err != nil {
		return fmt.Errorf("upload document object: %w", err)
	}
	if _, _, err := attachmentStore.Complete(ctx, principal, key("attach-doc-complete"), attachment.ID, attachmentapp.CompleteUpload{}); err != nil {
		return fmt.Errorf("complete attachment: %w", err)
	}
	if _, _, err := knowledgeStore.RequestIngestion(ctx, principal, key("ingest-1"), revision.ID, knowledgeapp.RequestIngestion{AttachmentID: attachment.ID}); err != nil {
		return fmt.Errorf("request ingestion: %w", err)
	}
	processor := &documentIngestor{pool: pool, objects: objectStore, cfgDocuments: cfg.Documents}
	if err := processor.ProcessDocument(ctx, revision.ID); err != nil {
		return fmt.Errorf("process document: %w", err)
	}
	// Re-load the revision because ingestion bumped its version.
	document, err := knowledgeStore.GetDocument(ctx, principal, doc.ID)
	if err != nil {
		return fmt.Errorf("reload document: %w", err)
	}
	freshRevision := document.Revisions[0]
	if _, _, err := knowledgeStore.ApproveRevision(ctx, principal, key("approve-doc"), revision.ID, knowledgeapp.ApproveRevision{ExpectedVersion: freshRevision.Version}); err != nil {
		return fmt.Errorf("approve document: %w", err)
	}

	// Step 5: copilot recommendation + correction feedback
	router := skawld.Router{Providers: structuredProviders}
	knowledgeService := knowledgeapp.Service{Store: knowledgeStore, Authorities: authorityReader, Now: clock.System{}.Now}
	recommendation, _, err := copilotapp.Service{
		Store: copilotStore, Search: knowledgeService, Router: router,
	}.Generate(ctx, principal, key("rec-1"), incident.ID, copilotapp.GenerateRecommendation{
		Question: "What is the next safe non-intrusive inspection step?",
	})
	if err != nil {
		return fmt.Errorf("generate recommendation: %w", err)
	}
	if err := (copilotapp.Service{Store: copilotStore}).Feedback(ctx, principal, key("fb-1"), recommendation.ID, copilotapp.Feedback{
		Outcome: "CORRECTED", Correction: "Also inspect coupling alignment", Reason: "Coupling was recently disturbed",
	}); err != nil {
		return fmt.Errorf("recommendation feedback: %w", err)
	}

	// Step 6: report draft/submit/approve
	reportService := reportapp.Service{
		Store: reportStore, Search: knowledgeService, Router: router,
		Authorities: authorityReader, Now: clock.System{}.Now,
	}
	report, _, err := reportService.Draft(ctx, principal, key("report-draft-1"), execution.ID)
	if err != nil {
		return fmt.Errorf("draft report: %w", err)
	}
	if _, _, err := (reportapp.Service{Store: reportStore}).Submit(ctx, principal, key("report-submit-1"), report.ID, reportapp.Transition{ExpectedVersion: report.Version}); err != nil {
		return fmt.Errorf("submit report: %w", err)
	}
	report, err = (reportapp.Service{Store: reportStore}).Get(ctx, principal, report.ID)
	if err != nil {
		return err
	}
	if _, _, err := reportService.Approve(ctx, principal, key("report-approve-1"), report.ID, reportapp.Transition{ExpectedVersion: report.Version}); err != nil {
		return fmt.Errorf("approve report: %w", err)
	}

	// Step 7: shift handover prepare/submit/accept
	handoverService := handoverapp.Service{
		Store: handoverStore, Router: router,
		Authorities: authorityReader, Now: clock.System{}.Now,
	}
	handover, _, err := handoverService.Prepare(ctx, principal, key("handover-1"), handoverapp.PrepareDraft{
		SiteID: siteID, ShiftStart: now.Add(-12 * time.Hour), ShiftEnd: now,
	})
	if err != nil {
		return fmt.Errorf("prepare handover: %w", err)
	}
	if _, _, err := (handoverapp.Service{Store: handoverStore}).Submit(ctx, principal, key("handover-submit-1"), handover.ID, handoverapp.Transition{ExpectedVersion: handover.Version}); err != nil {
		return fmt.Errorf("submit handover: %w", err)
	}
	handover, err = (handoverapp.Service{Store: handoverStore}).Get(ctx, principal, handover.ID)
	if err != nil {
		return err
	}
	if _, _, err := handoverService.Accept(ctx, principal, key("handover-accept-1"), handover.ID, handoverapp.Transition{ExpectedVersion: handover.Version}); err != nil {
		return fmt.Errorf("accept handover: %w", err)
	}

	// Step 8: two reviewed pump demonstrations
	demoGateway := skawld.DemonstrationGateway{Pool: pool, IDs: gen, Clock: c, Audit: audit.Sink{}}
	capture := skawld.CaptureProcessor{Pool: pool, Clock: c, Store: skawld.ObservationStore{Pool: pool, Clock: c}}
	demoService := demonstrationapp.Service{Gateway: demoGateway, Authorities: authorityReader, Now: clock.System{}.Now}
	var demos []demonstrationapp.Demonstration
	for _, label := range []string{"first", "second"} {
		demo, err := demoService.Start(ctx, principal, demonstrationapp.Start{
			SubjectKind: demonstrationapp.SubjectExecution, SubjectID: execution.ID,
		})
		if err != nil {
			return fmt.Errorf("start demonstration %s: %w", label, err)
		}
		base := time.Now().UTC()
		recommendationID := uuid.NewString()
		correction := "Inspect coupling alignment"
		if label == "first" {
			correction = "Inspect pump and motor alignment indicators"
		}
		for _, event := range []events.Event{
			{ID: uuid.NewString(), OrganizationID: organizationID, SiteID: siteID, ActorID: principal.ID,
				Type: "measurement.recorded", AggregateType: "measurement", AggregateID: uuid.NewString(),
				AggregateVersion: 1, SubjectKind: "EXECUTION", SubjectID: execution.ID,
				Payload:    map[string]any{"value": map[string]any{"measurement_type": "VIBRATION_VELOCITY", "value": "8.1", "unit": "MM_PER_S"}},
				OccurredAt: base},
			{ID: uuid.NewString(), OrganizationID: organizationID, SiteID: siteID, ActorID: principal.ID,
				Type: "copilot.recommendation.generated", AggregateType: "recommendation", AggregateID: recommendationID,
				AggregateVersion: 1, SubjectKind: "EXECUTION", SubjectID: execution.ID,
				Payload:    map[string]any{"value": map[string]any{"recommendation": "Inspect lubrication condition"}},
				OccurredAt: base.Add(time.Millisecond)},
			{ID: uuid.NewString(), OrganizationID: organizationID, SiteID: siteID, ActorID: principal.ID,
				Type: "copilot.recommendation.reviewed", AggregateType: "recommendation_feedback", AggregateID: uuid.NewString(),
				AggregateVersion: 1, SubjectKind: "EXECUTION", SubjectID: execution.ID,
				Payload: map[string]any{"recommendation_id": recommendationID, "value": map[string]any{
					"outcome": "CORRECTED", "correction": correction, "reason": "Coupling was recently disturbed " + label,
				}}, OccurredAt: base.Add(2 * time.Millisecond)},
			{ID: uuid.NewString(), OrganizationID: organizationID, SiteID: siteID, ActorID: principal.ID,
				Type: "execution.completed", AggregateType: "maintenance_execution", AggregateID: execution.ID,
				AggregateVersion: 2, SubjectKind: "EXECUTION", SubjectID: execution.ID,
				Payload:    map[string]any{"outcome_summary": "Alignment corrected " + label},
				OccurredAt: base.Add(3 * time.Millisecond)},
		} {
			if _, err := database.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) (struct{}, error) {
				return struct{}{}, events.Append(ctx, tx, event)
			}); err != nil {
				return fmt.Errorf("append demo event %s: %w", label, err)
			}
		}
		if err := capture.ProcessDemonstration(ctx, demo.ID, 100); err != nil {
			return fmt.Errorf("capture demo %s: %w", label, err)
		}
		demo, err = demoService.Complete(ctx, principal, demo.ID, demonstrationapp.Complete{Outcome: "completed " + label})
		if err != nil {
			return fmt.Errorf("complete demo %s: %w", label, err)
		}
		if _, err := demoService.Review(ctx, principal, demo.ID, demonstrationapp.Review{
			Decision: "APPROVED", Reason: "semantic trace reviewed for workflow learning",
		}); err != nil {
			return fmt.Errorf("review demo %s: %w", label, err)
		}
		demos = append(demos, demo)
	}

	// Step 9: compile/review/publish workflow
	workflowService := workflowapp.Service{
		Gateway:     skawld.WorkflowLearningGateway{Pool: pool, IDs: gen, Clock: c, Audit: audit.Sink{}},
		Authorities: authorityReader, Now: clock.System{}.Now,
	}
	candidate, err := workflowService.Compile(ctx, principal, workflowapp.Compile{
		Name:             "High vibration centrifugal pump inspection",
		Description:      "Compiled from two reviewed semantic pump demonstrations",
		DemonstrationIDs: []string{demos[0].ID, demos[1].ID},
	})
	if err != nil {
		return fmt.Errorf("compile workflow: %w", err)
	}
	effectiveAt := time.Now().UTC().Add(-time.Minute)
	approved, err := workflowService.Review(ctx, principal, candidate.WorkflowID, candidate.Version, workflowapp.Review{
		Decision: "APPROVED", Reason: "two coherent traces and safe guidance tools reviewed",
		Applicability: []workflowapp.Applicability{{
			SiteID: siteID, AssetID: asset.ID, AssetClass: "CENTRIFUGAL_PUMP",
			ValidationStatus: "VALIDATED",
		}},
		Prerequisites:        []string{"ENERGY_ISOLATION_WHEN_INTRUSIVE"},
		RequiredCompetencies: []string{"CENTRIFUGAL_PUMP"},
		EffectiveAt:          effectiveAt, ReviewAt: effectiveAt.AddDate(1, 0, 0),
	})
	if err != nil {
		return fmt.Errorf("review workflow: %w", err)
	}
	_ = approved
	if _, err := workflowService.Publish(ctx, principal, candidate.WorkflowID, candidate.Version, workflowapp.Publish{Reason: "evaluation gates and authority verified"}); err != nil {
		return fmt.Errorf("publish workflow: %w", err)
	}

	slogger.Info("demo data ready",
		"organization_id", organizationID, "site_id", siteID, "principal_id", principal.ID,
		"asset_id", asset.ID, "incident_id", incident.ID, "execution_id", execution.ID,
		"recommendation_id", recommendation.ID, "report_id", report.ID, "handover_id", handover.ID,
		"workflow_id", candidate.WorkflowID, "workflow_version", candidate.Version,
	)
	return nil
}

func seedFoundation(ctx context.Context, pool *pgxpool.Pool, gen id.Generator, clockRef clock.System) (identitydomain.Principal, string, string) {
	now := clockRef.Now()
	const subject = "seed-admin"

	// Idempotent: reuse the existing seed organization/site/principal.
	var principalID, organizationID, siteID string
	err := pool.QueryRow(ctx, `
		SELECT p.id::text,
		       coalesce(
		         (SELECT m.organization_id::text FROM memberships m
		          WHERE m.principal_id = p.id AND m.role = 'Administrator' LIMIT 1),
		         ''
		       ),
		       coalesce(
		         (SELECT m.site_id::text FROM memberships m
		          WHERE m.principal_id = p.id AND m.role = 'Administrator' LIMIT 1),
		         ''
		       )
		FROM principals p
		WHERE p.external_subject = $1
	`, subject).Scan(&principalID, &organizationID, &siteID)
	if err == nil && principalID != "" {
		if organizationID == "" {
			organizationID = gen.New()
			if _, err := pool.Exec(ctx, `
				INSERT INTO organizations (
					id, name, source_of_truth, version, created_at, updated_at
				) VALUES ($1::uuid, 'Demo Maintenance Co.', 'OWNED_BY_SKAWLD', 1, $2, $2)
			`, organizationID, now); err != nil {
				panic(fmt.Sprintf("insert organization: %v", err))
			}
			_, err = pool.Exec(ctx, `
				UPDATE memberships SET organization_id = $2::uuid
				WHERE principal_id = $1::uuid AND role = 'Administrator'
			`, principalID, organizationID)
			if err != nil {
				panic(fmt.Sprintf("update membership org: %v", err))
			}
		}
		if siteID == "" {
			siteID = gen.New()
			if _, err := pool.Exec(ctx, `
				INSERT INTO sites (
					id, organization_id, code, name, timezone, status, version, created_at, updated_at
				) VALUES ($1::uuid, $2::uuid, 'PLANT-A', 'Plant A', 'UTC', 'ACTIVE', 1, $3, $3)
			`, siteID, organizationID, now); err != nil {
				panic(fmt.Sprintf("insert site: %v", err))
			}
		}
		ensureSeedMembershipsAndAuthority(ctx, pool, gen, principalID, organizationID, siteID, now)
		return principalForSeed(principalID, organizationID, siteID), organizationID, siteID
	}

	principalID = gen.New()
	organizationID = gen.New()
	siteID = gen.New()
	if _, err := pool.Exec(ctx, `
		INSERT INTO organizations (
			id, name, source_of_truth, version, created_at, updated_at
		) VALUES ($1::uuid, 'Demo Maintenance Co.', 'OWNED_BY_SKAWLD', 1, $2, $2)
	`, organizationID, now); err != nil {
		panic(fmt.Sprintf("insert organization: %v", err))
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO sites (
			id, organization_id, code, name, timezone, status, version, created_at, updated_at
		) VALUES ($1::uuid, $2::uuid, 'PLANT-A', 'Plant A', 'UTC', 'ACTIVE', 1, $3, $3)
	`, siteID, organizationID, now); err != nil {
		panic(fmt.Sprintf("insert site: %v", err))
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO principals (
			id, external_subject, display_name, email, status, created_at, updated_at
		) VALUES ($1::uuid, 'seed-admin', 'Demo Administrator', 'admin@demo.example.invalid', 'ACTIVE', $2, $2)
		ON CONFLICT (external_subject) DO NOTHING
	`, principalID, now); err != nil {
		panic(fmt.Sprintf("insert principal: %v", err))
	}
	ensureSeedMembershipsAndAuthority(ctx, pool, gen, principalID, organizationID, siteID, now)
	return principalForSeed(principalID, organizationID, siteID), organizationID, siteID
}

func ensureSeedMembershipsAndAuthority(
	ctx context.Context,
	pool *pgxpool.Pool,
	gen id.Generator,
	principalID, organizationID, siteID string,
	now time.Time,
) {
	if _, err := pool.Exec(ctx, `
		INSERT INTO memberships (
			id, principal_id, organization_id, site_id, role, created_at
		) VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, 'Administrator', $5)
		ON CONFLICT (principal_id, organization_id, site_id, role) DO NOTHING
	`, gen.New(), principalID, organizationID, siteID, now); err != nil {
		panic(fmt.Sprintf("insert membership: %v", err))
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO approval_authorities (
			id, subject_id, organization_id, site_id, scope_kind, scope_id,
			competency, maximum_risk, valid_from, valid_until, delegated_by_subject_id, created_at
		) VALUES (
			$1::uuid, $2::uuid, $3::uuid, $4::uuid, NULL, NULL,
			NULL, 3, $5, NULL, NULL, $5
		)
	`, gen.New(), principalID, organizationID, siteID, now); err != nil {
		panic(fmt.Sprintf("insert approval authority: %v", err))
	}
}

func principalForSeed(principalID, organizationID, siteID string) identitydomain.Principal {
	principal := identitydomain.Principal{
		ID: principalID, OrganizationID: organizationID, SiteIDs: []string{siteID},
		Permissions: map[identitydomain.Permission]struct{}{},
		Roles:       []identitydomain.Role{identitydomain.RoleAdministrator},
	}
	for _, permission := range identitydomain.PermissionsForRole(identitydomain.RoleAdministrator) {
		principal.Permissions[permission] = struct{}{}
	}
	return principal
}

// seedRoleAccounts creates one principal per product role and binds it to the
// demo organization and site, so every role can be tested through the normal
// OIDC login flow. The OIDC subject matches the Keycloak dev realm users
// (deployments/compose/keycloak/skawld-realm.json). Idempotent on re-run.
func seedRoleAccounts(
	ctx context.Context,
	pool *pgxpool.Pool,
	gen id.Generator,
	organizationID, siteID string,
	now time.Time,
) {
	type roleAccount struct {
		subject string
		name    string
		email   string
		role    identitydomain.Role
	}
	accounts := []roleAccount{
		{subject: "00000000-0000-4000-8000-000000000101", name: "Dev Administrator", email: "admin@demo.example.invalid", role: identitydomain.RoleAdministrator},
		{subject: "00000000-0000-4000-8000-000000000102", name: "Dev Supervisor", email: "supervisor@demo.example.invalid", role: identitydomain.RoleMaintenanceSupervisor},
		{subject: "00000000-0000-4000-8000-000000000103", name: "Dev Senior Technician", email: "senior@demo.example.invalid", role: identitydomain.RoleSeniorTechnician},
		{subject: "00000000-0000-4000-8000-000000000104", name: "Dev Technician", email: "technician@demo.example.invalid", role: identitydomain.RoleTechnician},
		{subject: "00000000-0000-4000-8000-000000000105", name: "Dev Manager", email: "manager@demo.example.invalid", role: identitydomain.RoleManager},
	}
	for _, account := range accounts {
		principalID := gen.New()
		if err := pool.QueryRow(ctx, `
			INSERT INTO principals (
				id, external_subject, display_name, email, status, created_at, updated_at
			) VALUES ($1::uuid, $2, $3, $4, 'ACTIVE', $5, $5)
			ON CONFLICT (external_subject) DO UPDATE
			SET display_name = EXCLUDED.display_name,
			    email = EXCLUDED.email,
			    updated_at = EXCLUDED.updated_at
			RETURNING id::text
		`, principalID, account.subject, account.name, account.email, now).Scan(&principalID); err != nil {
			panic(fmt.Sprintf("seed role principal %s: %v", account.subject, err))
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO memberships (
				id, principal_id, organization_id, site_id, role, created_at
			) VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6)
			ON CONFLICT (principal_id, organization_id, site_id, role) DO NOTHING
		`, gen.New(), principalID, organizationID, siteID, string(account.role), now); err != nil {
			panic(fmt.Sprintf("seed role membership %s: %v", account.subject, err))
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO approval_authorities (
				id, subject_id, organization_id, site_id, scope_kind, scope_id,
				competency, maximum_risk, valid_from, valid_until, delegated_by_subject_id, created_at
			) VALUES (
				$1::uuid, $2::uuid, $3::uuid, $4::uuid, NULL, NULL,
				NULL, 3, $5, NULL, NULL, $5
			)
		`, gen.New(), principalID, organizationID, siteID, now); err != nil {
			panic(fmt.Sprintf("seed role authority %s: %v", account.subject, err))
		}
		slogger.Info("seeded role account", "role", account.role, "subject", account.subject)
	}
}

func key(label string) string {
	return fmt.Sprintf("seed-%s-%s", label, uuid.NewString()[:8])
}

// documentIngestor adapts the ingest.Processor to the jobs.DocumentIngestor
// contract so the seed can both enqueue and synchronously process documents.
type documentIngestor struct {
	pool         *pgxpool.Pool
	objects      objectstore.Store
	cfgDocuments config.Documents
}

func (d *documentIngestor) ProcessDocument(ctx context.Context, revisionID string) error {
	_, embeddingProvider, err := skawld.BuildProviders(skawld.AIConfig{}, nil)
	if err != nil {
		return fmt.Errorf("AI provider: %w", err)
	}
	processor := ingest.Processor{
		Pool: d.pool, Objects: d.objects,
		Extractor:  ingest.BoundedExtractor{PDFToTextBinary: d.cfgDocuments.PDFToTextBinary},
		Embeddings: embeddingProvider,
		IDs:        id.UUID{}, Clock: clock.System{}, Audit: audit.Sink{},
	}
	return processor.ProcessDocument(ctx, revisionID)
}

func lookupAsset(ctx context.Context, pool *pgxpool.Pool, organizationID, siteID, tag string) (assetapp.Asset, bool, error) {
	var value assetapp.Asset
	var id string
	err := pool.QueryRow(ctx, `
		SELECT id::text
		FROM assets
		WHERE organization_id = $1::uuid AND site_id = $2::uuid AND tag = $3
	`, organizationID, siteID, tag).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return value, false, nil
	}
	if err != nil {
		return value, false, err
	}
	// Reuse just the identity: components/criticality are set below anyway.
	value.ID = id
	value.SiteID = siteID
	value.Tag = tag
	return value, true, nil
}

func hasCriticality(ctx context.Context, pool *pgxpool.Pool, organizationID, assetID string) bool {
	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM asset_criticalities
			WHERE organization_id = $1::uuid AND asset_id = $2::uuid
		)
	`, organizationID, assetID).Scan(&exists); err != nil {
		return false
	}
	return exists
}

func lookupIncident(ctx context.Context, pool *pgxpool.Pool, organizationID, summary string) (incidentapp.Incident, bool, error) {
	var value incidentapp.Incident
	err := pool.QueryRow(ctx, `
		SELECT id::text, site_id::text, asset_id::text
		FROM incidents
		WHERE organization_id = $1::uuid AND summary = $2
		ORDER BY created_at ASC
		LIMIT 1
	`, organizationID, summary).Scan(&value.ID, &value.SiteID, &value.AssetID)
	if errors.Is(err, pgx.ErrNoRows) {
		return value, false, nil
	}
	if err != nil {
		return value, false, err
	}
	return value, true, nil
}

func lookupExecution(ctx context.Context, pool *pgxpool.Pool, organizationID, incidentID string) (executionapp.Execution, bool, error) {
	var id, state string
	err := pool.QueryRow(ctx, `
		SELECT id::text, state
		FROM maintenance_executions
		WHERE organization_id = $1::uuid AND incident_id = $2::uuid
		ORDER BY created_at ASC
		LIMIT 1
	`, organizationID, incidentID).Scan(&id, &state)
	if errors.Is(err, pgx.ErrNoRows) {
		return executionapp.Execution{}, false, nil
	}
	if err != nil {
		return executionapp.Execution{}, false, err
	}
	return executionapp.Execution{ID: id, State: state}, true, nil
}
