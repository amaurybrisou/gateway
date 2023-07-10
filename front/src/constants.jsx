const dev = { url: "http://localhost:3000/api" };
const prod = { url: "https://gw.puzzledge.org" };

const url = process.env.NODE_ENV === "production" ? prod.url : dev.url;

// const constants = {
const ACCESS_TOKEN_KEY = "access-token";
const API_URL = url;
// };

export { ACCESS_TOKEN_KEY, API_URL };
