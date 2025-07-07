import { useState } from 'react';
import { ExampleExportedFunc } from "../wailsjs/go/main/App";

function App() {
    const [resultText, setResultText] = useState("");
    const greet = () => ExampleExportedFunc("foo").then((result) => setResultText(result));

    return (
        <div>
            <h1>{resultText}</h1>
            <button onClick={greet}>click me!</button>
        </div>
    )
}

export default App;
