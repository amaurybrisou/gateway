const dev = { url: "http://localhost:8089" };
const prod = { url: "https://gw.puzzledge.org" };

export const ACCESS_TOKEN_KEY = "access-token";
export const API_URL = process.env.NODE_ENV === "production" ? prod.url: dev.url;
