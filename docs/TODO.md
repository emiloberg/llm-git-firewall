Planera

Det här är ett helt nytt projekt. Jag vill skapa ett helt nytt verktyg.

## Bakgrund

Jag kör Claude Code i en VM. Den kör med bypass permissions. Men jag vill inte ge den rättigheter att pusha mot Github. Jag vill skapa en allow-list på vilka git-kommandon som är tillåtna att köras.

VM:en (guest) och host delar en katalog (~/code/shared). Dvs guest kan skriva till en specifik katalog som innehåller alla projekt som Claude körs på.

Claude och guesten kan jobba som vanligt med de olika repona. Git är installerat. Men Github är inte autentiserat (guesten har ingen key). Så den kan inte pusha.

När Claude i guesten vill pusha data så kommer den att skapa en fil under katalogen `.git-llm-guard/pending`, alltså t.ex. `~/code/shared/my-org/project-a/.git-llm-guard/pending`.

Filen ska ha datum/tid som namn, och innehållet i den ska vara gitkommandot som ska köras. T.ex. `git push origin my-feature-branch`.

Den här appen, som ska köra på hosten, ska ligga och scanna och upptäcka om en ny fil skapas. Om det behövs av performance-skäl så ska den max kolla 2 kataloger djupt under root-katalogen (~/code/shared i mitt fall).

När den märker av en ny fil så ska den utföra kommandot på hosten.

MEN! Det ska finnas en konfigurerbar allow-list på vad som faktiskt tillåts. Som default så vill jag bara tillåta att vi pushar till brancher. Men inte till branchen som heter "main", "master", "develop". Dvs AI:n ska bara kunna pusha till en utvecklingsbranch. Allting annat ska INTE tillåtas.

När ett kommando är utfört så ska det skrivas ner till en logfil. Både vilket kommando som utfördes och outputen av kommandot.

Jag vill att det här ska kunna köras som en service/daemon (någonting headless) i slutändan. Kanske Go är ett bra språk för det?
