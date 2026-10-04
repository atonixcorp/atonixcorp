"use client";

import "../styles/HomePage.css";

// Top Ribbon
import TopHeader from "../sections/header/TopHeader";

// Main Header
import GlobalHeader from "../sections/header/GlobalHeader";

// Sections
import HeroSection from "../sections/hero/HeroSection";
import CoreCapabilities from "../sections/capabilities/CoreCapabilities";
import OperationalRail from "../sections/operational/OperationalRail";
import PlatformOverview from "../sections/platform/PlatformOverview";
import MissionSection from "../sections/mission/MissionSection";
import TechnologySovereignty from "../sections/sovereignty/TechnologySovereignty";
import StrategicRoadmap from "../sections/roadmap/StrategicRoadmap";
import Footer from "../sections/footer/Footer";

export default function HomePage() {
  return (
    <main className="home-page">

      <TopHeader />
      <GlobalHeader />

      <HeroSection />
      <CoreCapabilities />
      <OperationalRail />
      <PlatformOverview />
      <MissionSection />
      <TechnologySovereignty />
      <StrategicRoadmap />
    

      <Footer />
    </main>
  );
}
