import { useState } from "react";

function App() {
	const [message, setMessage] = useState<string>("");

	const fetchPublicData = () => {
		fetch(`http://localhost:${import.meta.env.VITE_PORT}/`, {
			credentials: "include",
		})
			.then((response) => response.text())
			.then((data) => setMessage(data))
			.catch((error) => console.error("Error fetching data:", error));
	};

	const fetchProtectedData = () => {
		fetch(`http://localhost:${import.meta.env.VITE_PORT}/protected`, {
			credentials: "include",
		})
			.then((response) => response.text())
			.then((data) => setMessage(data))
			.catch((error) => console.error("Error fetching data:", error));
	};

	const login = () => {
		fetch(`http://localhost:${import.meta.env.VITE_PORT}/login`, {
			credentials: "include",
		})
			.then((response) => response.text())
			.then((data) => setMessage(data))
			.catch((error) => console.error("Error fetching data:", error));
	};

	const register = () => {
		fetch(`http://localhost:${import.meta.env.VITE_PORT}/register`, {
			credentials: "include",
		})
			.then((response) => response.text())
			.then((data) => setMessage(data))
			.catch((error) => console.error("Error fetching data:", error));
	};

	const logout = () => {
		fetch(`http://localhost:${import.meta.env.VITE_PORT}/logout`, {
			credentials: "include",
		})
			.then((response) => response.text())
			.then((data) => setMessage(data))
			.catch((error) => console.error("Error fetching data:", error));
	};

	return (
		<div className="min-h-screen bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
			<div className=" mx-auto space-y-8 w-full max-w-3xl">
				<div className="text-center">
					<h1 className="text-4xl font-bold text-gray-900 mb-2">
						Welcome to Vite + React + Go
					</h1>
					<p className="text-gray-600">
						Get started by editing{" "}
						<code className="text-sm bg-gray-100 p-1 rounded">
							src/App.tsx
						</code>
					</p>
				</div>

				<div className="bg-white p-6 rounded-lg shadow-md">
					<div className="text-center space-y-4 flex flex-col items-center justify-center">
						<div className="space-x-4">
							<button
								onClick={fetchPublicData}
								className="bg-green-500 hover:bg-green-600 text-white font-semibold py-2 px-4 rounded-md transition-colors"
							>
								Fetch public endpoint
							</button>

							<button
								onClick={fetchProtectedData}
								className="bg-fuchsia-500 hover:bg-fuchsia-600 text-white font-semibold py-2 px-4 rounded-md transition-colors"
							>
								Fetch protected endpoint
							</button>
						</div>

						{message && (
							<div className="w-full mt-4 p-4 bg-gray-50 rounded-md text-left">
								<p className="text-gray-700">
									Server Response:
								</p>
								<p className="text-gray-900 font-medium whitespace-pre-line">
									{message}
								</p>
							</div>
						)}

						<div className="space-x-4">
							<button
								onClick={register}
								className="bg-purple-500 hover:bg-purple-600 text-white font-semibold py-2 px-4 rounded-md transition-colors"
							>
								Sign In
							</button>

							<button
								onClick={login}
								className="bg-sky-500 hover:bg-sky-600 text-white font-semibold py-2 px-4 rounded-md transition-colors"
							>
								Login
							</button>

							<button
								onClick={logout}
								className="bg-rose-500 hover:bg-rose-600 text-white font-semibold py-2 px-4 rounded-md transition-colors"
							>
								Logout
							</button>
						</div>
					</div>
				</div>

				<div className="text-center text-gray-500 text-sm">
					Built with Vite, React, Go, and Tailwind CSS
				</div>
			</div>
		</div>
	);
}

export default App;
