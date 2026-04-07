/** Base URL de l’API (gateway), ex. https://api-gateway.xxx.azurecontainerapps.io/api/v1 */
export const API_BASE_URL =
  process.env.REACT_APP_API_BASE_URL ||
  process.env.REACT_APP_API_URL_AUTH ||
  "http://localhost:8080/api/v1";
