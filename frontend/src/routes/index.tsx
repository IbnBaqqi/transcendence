import { BrowserRouter, Routes, Route } from "react-router-dom";
import Home from "../pages/Home";

export default function AppRouter() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/privacy" element={<Home />} />
        <Route path="/terms" element={<Home />} />
      </Routes>
    </BrowserRouter>
  );
}