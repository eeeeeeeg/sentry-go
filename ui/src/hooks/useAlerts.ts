import { createWebhookAlert, listAlertDeliveriesPage, listAlerts, testAlert, updateAlertStatus, type AlertDelivery, type AlertRule, type Paginated } from "../services/api";
import { useAsyncData } from "./useAsyncData";

const emptyDeliveries: Paginated<AlertDelivery> = {
  items: [],
  page: { limit: 20, offset: 0, total: 0 },
};

export function useAlerts(projectId: string, refreshKey = 0, deliveryOffset = 0, deliveryLimit = 20, deliveryStatus = "") {
  const state = useAsyncData(
    async () => {
      const [alerts, deliveries] = await Promise.all([
        listAlerts(projectId),
        listAlertDeliveriesPage(projectId, { status: deliveryStatus || undefined, limit: deliveryLimit, offset: deliveryOffset }),
      ]);
      return { alerts, deliveries };
    },
    { alerts: [] as AlertRule[], deliveries: emptyDeliveries },
    [projectId, refreshKey, deliveryOffset, deliveryLimit, deliveryStatus],
  );

  async function createAlert(form: FormData) {
    await createWebhookAlert(projectId, {
      name: String(form.get("name") ?? ""),
      event_type: String(form.get("event_type") ?? "new_issue"),
      webhook_url: String(form.get("webhook_url") ?? ""),
      min_level: String(form.get("min_level") ?? "error"),
      threshold_count: Number(form.get("threshold_count") || 1),
      window_seconds: Number(form.get("window_seconds") || 300),
      cooldown_seconds: Number(form.get("cooldown_seconds") || 300),
    });
    await state.reload();
  }

  async function setAlertStatus(alertId: string, status: string) {
    await updateAlertStatus(alertId, status);
    await state.reload();
  }

  async function sendTestAlert(alertId: string) {
    await testAlert(alertId);
  }

  return { ...state, createAlert, setAlertStatus, sendTestAlert };
}
