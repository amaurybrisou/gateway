/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./src/**/*.{js,jsx,ts,tsx}", "./node_modules/flowbite/**/*.js"],
  theme: {
    extend: {
      colors: {
        primary: "#ff0000", // Replace with your desired color
      },
    },
  },
  plugins: [
    require('flowbite/plugin')
  ],
};
