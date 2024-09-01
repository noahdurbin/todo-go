# Todo app in Go and HTMX
![image](https://github.com/user-attachments/assets/ea23dde7-a635-45c0-a182-fa320573d136)

## stack
- Go
- HTMX
- Tailwind
- Mongodb

## Local Setup

To run this Todo application on your local machine, follow these steps:

1. **Prerequisites:**
   - Go (version 1.16 or later)
   - MongoDB Atlas account (or a local MongoDB installation)
   - Git

2. **Clone the repository:**
   ```
   git clone https://github.com/noahdurbin/todo-go.git
   cd todo-app
   ```

3. **Set up environment variables:**
   Create a `.env` file in the root directory of the project and add the following:
   ```
   MONGODB_URI=your_mongodb_connection_string
   PORT=8080
   ```
   Replace `your_mongodb_connection_string` with your actual MongoDB Atlas connection string or local MongoDB URI.

4. **Install dependencies:**
   ```
   go mod tidy
   ```

5. **Create the HTML template:**
   Create a file named `index.html` in the root directory and add your HTML template for the Todo app.

6. **Create a static folder:**
   Create a folder named `static` in the root directory for any static assets (CSS, JavaScript, images).

7. **Run the application:**
   ```
   go run main.go
   ```

8. **Access the application:**
   Open your web browser and navigate to `http://localhost:8080` (or whatever port you specified in the .env file).

## Notes:
- Ensure that your MongoDB Atlas cluster (or local MongoDB instance) is running and accessible.
- If you're using MongoDB Atlas, make sure to whitelist your IP address in the Atlas dashboard.
- The application will create a database named `tododb` and a collection named `todos` automatically.

If you encounter any issues, ensure all dependencies are correctly installed and that your MongoDB connection string is correct.
