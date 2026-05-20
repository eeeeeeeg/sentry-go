import axios, { AxiosError, AxiosRequestConfig } from "axios";

export type ApiErrorPayload = {
  error?: string;
  message?: string;
};

export const http = axios.create({
  baseURL: "/",
  timeout: 10000,
  headers: {
    "Content-Type": "application/json",
  },
});

http.interceptors.request.use((config) => {
  const token = localStorage.getItem("sentry-lite-token");
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

http.interceptors.response.use(
  (response) => response,
  (error: AxiosError<ApiErrorPayload>) => {
    const status = error.response?.status;
    const payload = error.response?.data;
    const message = payload?.message || payload?.error || error.message || "请求失败";

    if (status === 401) {
      window.dispatchEvent(new CustomEvent("sentry-lite:unauthorized"));
    }

    return Promise.reject(new Error(message));
  },
);

export async function request<T>(config: AxiosRequestConfig): Promise<T> {
  const response = await http.request<T>(config);
  return response.data;
}
