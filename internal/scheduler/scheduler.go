package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/monocle-dev/monocle/db"
	"github.com/monocle-dev/monocle/internal/models"
	"github.com/monocle-dev/monocle/internal/monitors"
	"github.com/monocle-dev/monocle/internal/services"
	"github.com/monocle-dev/monocle/internal/types"
	"gorm.io/gorm"
)

type Scheduler struct {
	monitors map[uint]*MonitorJob // monitor ID -> job
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

type MonitorJob struct {
	monitor models.Monitor
	ticker  *time.Ticker
	cancel  context.CancelFunc
}

// NewScheduler initializes a new Scheduler instance
func NewScheduler() *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		monitors: make(map[uint]*MonitorJob),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start loads all active monitors and begins scheduling
func (s *Scheduler) Start() error {
	log.Println("Starting scheduler...")

	var monitorsList []models.Monitor
	if err := db.DB.Where("status = ?", "active").Find(&monitorsList).Error; err != nil {
		return err
	}

	for _, monitor := range monitorsList {
		s.AddMonitor(monitor)
	}

	log.Printf("Scheduler started with %d monitors", len(monitorsList))
	return nil
}

// Stop gracefully shuts down all monitor jobs
func (s *Scheduler) Stop() {
	log.Println("Stopping scheduler...")
	s.cancel() // Cancel main context

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, job := range s.monitors {
		job.ticker.Stop()
		job.cancel()
	}

	s.monitors = make(map[uint]*MonitorJob)
	log.Println("Scheduler stopped")
}

// AddMonitor starts monitoring for a specific monitor
func (s *Scheduler) AddMonitor(monitor models.Monitor) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Stop existing job if it exists
	if existingJob, exists := s.monitors[monitor.ID]; exists {
		existingJob.ticker.Stop()
		existingJob.cancel()
	}

	// Create new job
	jobCtx, jobCancel := context.WithCancel(s.ctx)
	ticker := time.NewTicker(time.Duration(monitor.Interval) * time.Second)

	job := &MonitorJob{
		monitor: monitor,
		ticker:  ticker,
		cancel:  jobCancel,
	}

	s.monitors[monitor.ID] = job

	// Start the monitoring goroutine with immediate check
	go func() {
		// Execute immediate check with a copy of monitor data
		monitorCopy := monitor
		s.executeCheck(monitorCopy)
		// Then start regular monitoring
		s.runMonitor(jobCtx, job)
	}()

	log.Printf("Added monitor %d (%s) with immediate check", monitor.ID, monitor.Name)
}

// RemoveMonitor stops monitoring for a specific monitor
func (s *Scheduler) RemoveMonitor(monitorID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if job, exists := s.monitors[monitorID]; exists {
		job.ticker.Stop()
		job.cancel()
		delete(s.monitors, monitorID)
		log.Printf("Removed monitor %d", monitorID)
	}
}

// UpdateMonitor updates an existing monitor (stops old, starts new)
func (s *Scheduler) UpdateMonitor(monitor models.Monitor) {
	s.AddMonitor(monitor) // AddMonitor handles stopping existing job
}

// runMonitor executes the actual monitoring logic
func (s *Scheduler) runMonitor(ctx context.Context, job *MonitorJob) {
	defer job.ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-job.ticker.C:
			// Get a safe copy of the monitor data under read lock
			s.mu.RLock()
			monitorCopy := job.monitor
			s.mu.RUnlock()

			s.executeCheck(monitorCopy)
		}
	}
}

// executeCheck performs the actual monitor check
func (s *Scheduler) executeCheck(monitor models.Monitor) {
	start := time.Now()
	var err error

	switch monitor.Type {
	case "http":
		var cfg types.HttpConfig
		if unmarshalErr := json.Unmarshal(monitor.Config, &cfg); unmarshalErr != nil {
			log.Printf("Invalid HTTP config for monitor %d: %v", monitor.ID, unmarshalErr)
			return
		}
		err = monitors.GetHTTP(&cfg)
	case "dns":
		var cfg types.DNSConfig

		if unmarshalErr := json.Unmarshal(monitor.Config, &cfg); unmarshalErr != nil {
			log.Printf("Invalid DNS config for monitor %d: %v", monitor.ID, unmarshalErr)
			return
		}

		err = monitors.CheckDNS(&cfg)
	case "database":
		var cfg types.DatabaseConfig

		if unmarshalErr := json.Unmarshal(monitor.Config, &cfg); unmarshalErr != nil {
			log.Printf("Invalid Database config for monitor %d: %v", monitor.ID, unmarshalErr)
			return
		}

		err = monitors.CheckDatabase(&cfg)
	default:
		log.Printf("Unsupported monitor type: %s", monitor.Type)
		return
	}

	responseTime := time.Since(start)
	s.storeCheckResult(monitor, err, responseTime)

	if err != nil {
		log.Printf("Monitor %d failed: %v", monitor.ID, err)
	} else {
		log.Printf("Monitor %d succeeded in %v", monitor.ID, responseTime)
	}
}

// storeCheckResult saves the check result to database
func (s *Scheduler) storeCheckResult(monitor models.Monitor, err error, responseTime time.Duration) {
	status := "success"
	message := ""

	now := time.Now()

	var activeIncident models.Incident

	if err := db.DB.Where("monitor_id = ? AND status = ?", monitor.ID, "active").First(&activeIncident).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			log.Printf("Failed to check for active incident for monitor %d: %v", monitor.ID, err)
		}
	}

	if err != nil {
		status = "failure"
		message = err.Error()

		if activeIncident.ID == 0 {
			// Create a new incident if none exists
			title := s.generateIncidentTitle(monitor)
			description := s.generateIncidentDescription(monitor, err)

			newIncident := models.Incident{
				MonitorID:   monitor.ID,
				Status:      "active",
				StartedAt:   &now,
				Title:       title,
				Description: description,
			}

			if projectErr := db.DB.Create(&newIncident).Error; projectErr != nil {
				log.Printf("Failed to create incident for monitor %d: %v", monitor.ID, err)
			} else {
				log.Printf("Created new incident for monitor %d", monitor.ID)

				var project models.Project
				if err := db.DB.First(&project, monitor.ProjectID).Error; err == nil {
					newIncident.Monitor = monitor
					if notifyErr := services.SendIncidentCreatedNotification(project, newIncident); notifyErr != nil {
						log.Printf("Failed to send incident created notification: %v", notifyErr)
					} else {
						log.Printf("Successfully sent incident created notification")
					}
				} else {
					log.Printf("Failed to load project for notification: %v", projectErr)
				}
			}
		}

	} else if activeIncident.ID != 0 {
		activeIncident.ResolvedAt = &now
		activeIncident.Status = "resolved"

		if projectErr := db.DB.Save(&activeIncident).Error; projectErr != nil {
			log.Printf("Failed to save active incident for monitor %d", monitor.ID)
		} else {
			log.Printf("Saved resolved active incident for monitor %d", monitor.ID)

			var project models.Project
			if err := db.DB.First(&project, monitor.ProjectID).Error; err == nil {
				activeIncident.Monitor = monitor
				if notifyErr := services.SendIncidentResolvedNotification(project, activeIncident); notifyErr != nil {
					log.Printf("Failed to send incident resolved notification: %v", notifyErr)
				}
			} else {
				log.Printf("Failed to load project for notification: %v", projectErr)
			}
		}
	}

	check := models.MonitorCheck{
		MonitorID:    monitor.ID,
		Status:       status,
		ResponseTime: int(responseTime.Milliseconds()),
		Message:      message,
		CheckedAt:    time.Now(),
	}

	if err := db.DB.Create(&check).Error; err != nil {
		log.Printf("Failed to store check result for monitor %d: %v", monitor.ID, err)
	}
}

// generateIncidentTitle creates a descriptive title for an incident
func (s *Scheduler) generateIncidentTitle(monitor models.Monitor) string {
	var title string

	switch monitor.Type {
	case "http":
		title = fmt.Sprintf("HTTP monitor '%s' is down", monitor.Name)
	case "dns":
		title = fmt.Sprintf("DNS monitor '%s' is failing", monitor.Name)
	case "database":
		title = fmt.Sprintf("Database monitor '%s' is unreachable", monitor.Name)
	default:
		title = fmt.Sprintf("Monitor '%s' (%s) is failing", monitor.Name, monitor.Type)
	}

	return title
}

// generateIncidentDescription creates a detailed description for an incident
func (s *Scheduler) generateIncidentDescription(monitor models.Monitor, err error) string {
	var description strings.Builder

	description.WriteString(fmt.Sprintf("Monitor '%s' has failed.\n\n", monitor.Name))

	// Add error details
	if err != nil {
		description.WriteString(fmt.Sprintf("Error: %s\n\n", err.Error()))
	}

	// Add monitor configuration details
	description.WriteString("Monitor Configuration:\n")
	description.WriteString(fmt.Sprintf("  Type: %s\n", monitor.Type))
	description.WriteString(fmt.Sprintf("  Check Interval: %d seconds\n", monitor.Interval))

	switch monitor.Type {
	case "http":
		var cfg types.HttpConfig
		if json.Unmarshal(monitor.Config, &cfg) == nil {
			description.WriteString(fmt.Sprintf("  URL: %s\n", cfg.URL))
			description.WriteString(fmt.Sprintf("  Method: %s\n", cfg.Method))
			description.WriteString(fmt.Sprintf("  Expected Status: %d\n", cfg.ExpectedStatus))
			if cfg.Timeout > 0 {
				description.WriteString(fmt.Sprintf("  Timeout: %d seconds\n", cfg.Timeout))
			}
		}
	case "dns":
		var cfg types.DNSConfig
		if json.Unmarshal(monitor.Config, &cfg) == nil {
			description.WriteString(fmt.Sprintf("  Domain: %s\n", cfg.Domain))
			if cfg.RecordType != "" {
				description.WriteString(fmt.Sprintf("  Record Type: %s\n", strings.ToUpper(cfg.RecordType)))
			}
			if cfg.Expected != "" {
				description.WriteString(fmt.Sprintf("  Expected Value: %s\n", cfg.Expected))
			}
		}
	case "database":
		var cfg types.DatabaseConfig
		if json.Unmarshal(monitor.Config, &cfg) == nil {
			description.WriteString(fmt.Sprintf("  Database Type: %s\n", strings.ToUpper(cfg.Type)))
			description.WriteString(fmt.Sprintf("  Host: %s:%d\n", cfg.Host, cfg.Port))
			description.WriteString(fmt.Sprintf("  Database: %s\n", cfg.Database))
			description.WriteString(fmt.Sprintf("  Username: %s\n", cfg.Username))
		}
	}

	return description.String()
}

// GetStatus returns current scheduler status
func (s *Scheduler) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"active_monitors": len(s.monitors),
		"running":         s.ctx.Err() == nil,
	}
}

// Global scheduler instance
var globalScheduler *Scheduler

// Initialize creates and starts the global scheduler
func Initialize() error {
	globalScheduler = NewScheduler()
	return globalScheduler.Start()
}

// Shutdown stops the global scheduler
func Shutdown() {
	if globalScheduler != nil {
		globalScheduler.Stop()
	}
}

// AddMonitor adds a monitor to the global scheduler
func AddMonitor(monitor models.Monitor) {
	if globalScheduler != nil {
		globalScheduler.AddMonitor(monitor)
	}
}

// RemoveMonitor removes a monitor from the global scheduler
func RemoveMonitor(monitorID uint) {
	if globalScheduler != nil {
		globalScheduler.RemoveMonitor(monitorID)
	}
}

// UpdateMonitor updates a monitor in the global scheduler
func UpdateMonitor(monitor models.Monitor) {
	if globalScheduler != nil {
		globalScheduler.UpdateMonitor(monitor)
	}
}
