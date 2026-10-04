"use client";

import "../../styles/MissionSection.css";

export default function MissionSection() {
  return (
    <section className="mission" id="mission">
      <div className="mission-wrap">

        {/* Left Heading */}
        <div className="mission-heading">
          <p className="mission-index">02 / Mission</p>

          <h2 className="mission-title">
            Build.<br />
            Protect.<br />
            <span className="mission-accent">Innovate.</span><br />
            Scale.
          </h2>
        </div>

        {/* Right Details */}
        <div className="mission-detail">
          <p className="mission-summary">
            We are building the cloud foundation for ambitious ideas and essential
            systems—supporting the people who research, engineer, and operate them.
          </p>

          <ul className="mission-list">
            <li>Enable AI, scientific discovery, and high-performance workloads</li>
            <li>Support engineering across robotics, aerospace, flight, and energy</li>
            <li>Bring compute, data, networking, and security into a connected foundation</li>
            <li>Scale critical applications from development through operation</li>
          </ul>
        </div>

      </div>
    </section>
  );
}
