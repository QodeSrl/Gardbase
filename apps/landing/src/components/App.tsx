import React from "react";
import { Route, Routes } from "react-router-dom";
import HomePage from "@/pages/MainPage";

const App: React.FC = () => {
  return (
    <Routes>
      <Route path="*" element={<HomePage />} />
    </Routes>
  );
};

export default App;
