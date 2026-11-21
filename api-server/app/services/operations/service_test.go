package operations

import (
	"api-server/v2/modelMethods/operations"
	"testing"
	"time"
)

func TestAlertsUseAgeThresholdsAndIgnoreEmptyQueues(t *testing.T) {
	if len(NotificationAlerts(operations.Snapshot{PendingNotifications: 1, OldestNotificationSeconds: 901}, 15*time.Minute)) != 1 {
		t.Fatal("old notification queue not alerted")
	}
	if len(NotificationAlerts(operations.Snapshot{OldestNotificationSeconds: 901}, 15*time.Minute)) != 0 {
		t.Fatal("empty notification queue alerted")
	}
	snapshot := operations.Snapshot{OpenReviews: 1, OldestReviewSeconds: 301, PendingRefunds: 1, OldestRefundSeconds: 86401, OverdueCheckouts: 1, OldestCheckoutSeconds: 901}
	if alerts := Alerts(snapshot, 5*time.Minute, 24*time.Hour, 15*time.Minute); len(alerts) != 3 {
		t.Fatal(alerts)
	}
	snapshot.OpenReviews = 0
	snapshot.PendingRefunds = 0
	snapshot.OverdueCheckouts = 0
	if alerts := Alerts(snapshot, 5*time.Minute, 24*time.Hour, 15*time.Minute); len(alerts) != 0 {
		t.Fatal("empty queues alerted", alerts)
	}
}
