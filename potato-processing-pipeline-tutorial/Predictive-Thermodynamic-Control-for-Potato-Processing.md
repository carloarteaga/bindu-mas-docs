**System Requirements Specification (SRS): Predictive Thermodynamic Control for Potato Processing**

**1. Executive Summary & Objective**
The objective of this specification is to define the mathematical and physical parameters for an automated control system designed to optimize the potato chip frying process. The system aims to mitigate the adverse effects of Cold-Induced Sweetening (CIS) by integrating real-time Computer Vision analysis with established kinetic and thermodynamic models. The final output of this logic will dynamically generate hardware setpoints for industrial Programmable Logic Controllers (PLCs) managing storage silos and frying machinery.

---

**2. Scientific Background: The Problem**
During post-harvest transport and storage, potatoes exposed to temperatures below 8°C–10°C undergo a physiological stress response known as Cold-Induced Sweetening (CIS). The enzyme invertase catalyzes the breakdown of starch into reducing sugars (glucose and fructose).

When potato slices with high reducing sugar concentrations are exposed to high frying temperatures (175°C–190°C), they undergo an accelerated Maillard reaction (a non-enzymatic browning between amino acids and reducing sugars). This results in unacceptable dark pigmentation, bitter flavors, and elevated acrylamide levels. To prevent this, the system must dynamically adjust storage times and frying parameters based on the incoming tuber's exact chemical state.

---

**3. Phase I: Optical Sensory Analysis (AI Vision Input)**
Traditional chemical titration to measure reducing sugars ($[RS]$) is destructive and time-consuming. This system will utilize an Optical AI Model as a virtual sensor to estimate the $[RS]$ concentration from raw potato slices.

* **Input:** High-resolution RGB images of raw, freshly sliced potatoes.
* **Visual Markers:** The model will analyze the structural integrity of the starch matrix. Starch-rich potatoes are highly opaque and uniform (white/cream). Sugar-rich potatoes exhibit water-soaked, translucent, or localized yellowish areas (often at the basal ends).
* **Output:** The model must output an **Estimated Reducing Sugar Coefficient ($C_{RS}$)**, a normalized float value between $0.0$ (Ideal/Low Sugar) and $1.0$ (Critical/High Sugar). This coefficient serves as the $[RS]_0$ proxy for the kinetic equations.

---

**4. Phase II: Mathematical Modeling & Control Logic**

Based on the $C_{RS}$ acquired from Phase I, the system must calculate the required PLC setpoints sequentially using the following models:

**Model A: Tuber Reconditioning (First-Order Kinetics)**
If the $C_{RS}$ exceeds acceptable limits ($> 0.2$), the potatoes must be stored in a climate-controlled silo (typically 15°C - 20°C) to allow cellular respiration to consume the excess sugars. The decay of reducing sugars follows first-order kinetics:

$$[RS]_t = [RS]_{eq} + \left( [RS]_0 - [RS]_{eq} \right) e^{-kt}$$

* **Goal:** Solve for $t$ (days required in storage).
* **Parameters:**
* $[RS]_0$: Initial sugar concentration derived from $C_{RS}$.
* $[RS]_t$: Target sugar concentration ($\le 0.2\%$).
* $[RS]_{eq}$: Basal equilibrium concentration of the specific potato variety.
* $k$: Respiration rate constant at the designated storage temperature.



**Model B: Frying Temperature Adjustment (Arrhenius Equation)**
If the potato must be processed before reaching optimal $[RS]_t$, or to fine-tune the final color, the fryer oil temperature ($T$) must be adjusted. The Maillard browning rate ($k_{color}$) must remain constant to guarantee product uniformity.

$$k_{color} = A \cdot [RS]^n \cdot \exp\left( \frac{-E_a}{R \cdot T} \right)$$

* **Goal:** Solve for $T$ (Target Oil Temperature in Kelvin).
* **Logic:** As $[RS]$ increases, the exponential term must decrease to maintain a constant $k_{color}$. Therefore, higher sugar dictates a lower frying temperature (e.g., dropping from 185°C to 165°C).
* **Parameters:**
* $A$: Pre-exponential factor.
* $E_a$: Activation energy for the Maillard reaction (approx. 100-150 kJ/mol).
* $R$: Universal gas constant (8.314 J/mol·K).



**Model C: Conveyor Belt / Frying Time (Fick’s Second Law of Diffusion)**
Because Model B may lower the oil temperature ($T$), the rate of water evaporation will decrease. To achieve the required final moisture content (approx. 1.5% for crispness) without burning the chip, the time the potato spends in the oil ($t_{fry}$) must be extended by adjusting the conveyor belt speed. Moisture reduction ($MR$) is modeled by:

$$MR = \frac{8}{\pi^2} \exp\left( \frac{-D_{eff} \cdot \pi^2 \cdot t_{fry}}{4L^2} \right)$$

* **Goal:** Solve for $t_{fry}$ (Required frying time).
* **Parameters:**
* $D_{eff}$: Effective moisture diffusivity (which is a function of the oil temperature $T$ calculated in Model B).
* $L$: Half-thickness of the potato slice.



---

**5. System Output Specifications (Hardware Setpoints)**
The evaluation of Models A, B, and C will yield the final deterministic values required to safely operate the processing line. The system will output these as discrete commands to the plant's hardware controllers:

1. **Silo_Temp_SP:** Setpoint for storage heater/cooler (°C).
2. **Silo_Duration_SP:** Calculated time $t$ for the batch to remain in quarantine/reconditioning (Days).
3. **Fryer_Oil_Temp_SP:** Modbus Write Command for the heat exchanger regulating oil temperature $T$ (°C).
4. **Fryer_Belt_Speed_SP:** Modbus Write Command for the VFD (Variable Frequency Drive) controlling the conveyor belt, derived from $t_{fry}$ (Hz or m/s).