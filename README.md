# Automatsko slanje poruka na elitemadzone.org

### Šta je EliteMadZone.org?

Elitesecurity je nekada bio veoma popularan IT forum, a danas više forum sa nostalgičnom i arhivskom vrednošću. Najveći deo korisnika je otišao na druge, modernije platforme sa većim brojem aktivnih korisnika, na primer Reddit ili Stack Overflow. 

ES je od samog početka imao i poseban odeljak za razgovor o netehničkim temama. U početku je to bio potforum u okviru glavnog sajta, a kasnije je preseljen na poseban domen, **elitemadzone.org**. U suštini to je mesto gde se najviše razgovara o politici, svetskoj i domaćoj, iako su pravilnikom foruma političke teme zabranjene. [Pravilnik](https://www.youtube.com/watch?v=k9ojK9Q_ARE) je nešto što bismo pre mogli nazvati smernicama, nego stvarnim pravilima. Nažalost i ovaj ćerka-forum je doživeo sličnu sudbinu kao i ES, zbog zastarelog koncepta i tehnologije, ali i zbog **neprincipijelne administracije**. 

### Let's Make EMZ Great Again
Srećom, tim programera koji čine korisnici foruma sa nadimcima Everovic, Zurg, Buzzlightyear i ja (Bora Sonar), rešili su da osavremene vremešni forum, tako što će napisati ovu predivnu aplikaciju za automatsko objavljivanje poruka. Najbolja stvar je što uopšte ne morate da imate web brauzer i otvarate forum da biste objavili poruku!

### Kako aplikacija radi?

Program na svakih 10 minuta proverava da li se Vaše objave nalaze na sajtu. Ukoliko se ne nalaze program ih objavljuje. Da bi funkcionisao potrebno je na napravite nekoliko jednostavnih podešavanja.

- Izmenite fajl nalozi.json tako što ćete uneti korisničko ime, lozinku i postaviti atribut aktivan na true. Ukoliko imate prijatelje sa kojim delite program, u fajl možete uneti i njihove parametre. Ako administrator greškom banuje neki od naloga, program to detektuje, deaktivira nalog postavljanjem atributa aktivan na false, a objave tog korisnika prebacuje nekom od aktivnih.
 - Pokrenite tekst editor, napišite tekst objave i sačuvajte je u bilo kom potfolderu koji se nalazi ispod direktorijuma programa. Pogledajte direktorijum *tekst* kao primer.
- Dodajte opis objave u fajl objave.json. **Fajl sadrži listu objava, koje su date kao primer unosa. Ukoliko ne želite da objavite primere izbrišite sadržaj fajla i dodajte svoje unose.** Za svaku objavu mora da postoji odgovarajući unos. Unos može da izgleda ovako:
```json
  {
      "naslov": "Moja prva poruka",
      "postId": "?",
      "temaId": "510739",
      "autor": "Pera Peric",
      "fajl": "tekst/srbija/srbija-prvi-test.txt"
  }
```
    - **naslov** pišete Vi sami, vodite računa da forum zahteva da naslov ima odgovarajuću dužinu (čini mi se da mora biti duži od 15 karaktera)
    - **postId** je identifikator Vaše objave. Ukoliko pišete novu objavu u polje možete da upišete bilo šta, na primer znak pitanja. Ukoliko se radi o već postojećoj objavi, onda je potrebno da pročitate postId iz adrese objave. Na primer, ako je adresa objave https://www.elitemadzone.org/t510739-36#4101985, postId je 4101985, a temaId 510739
    - **temaId** podatak mora biti tačan, kako bi objava završila na pravom mestu. Čita se iz adrese, na iznad prikazani način;
    - **autor**  je ime koje koristite prilikom prijave na sajt
    - **fajl** je relativna putanja do fajla koji sadrži tekst objave
- Nakon što ste uneli potrebne opise i sačuvali fajlove, možete pokrenuti program.

### Uslovi korišćenja

Program koristite na sopstvenu odgovornost i pod uslovima navedenim u MIT licenci. Kao što je navedeno u MIT licenci, ne dajem nikakve garancije i sa programom možete da radite šta god hoćete, menjate program, delite, prodate ga... Tehnički, legalni ili bilo koji drugi aspekti koji proizilaze iz korišćenja programa su isključivo odgovornost korisnika koji pokreće program ili koristi program na bilo koji način. Jedino zadržavam autorsko pravo. Autor programa je vlasnik imejl naloga borasonar@tutamail.com. 

Ukoliko imate dodatnih pitanja možete mi pisati na gore navedeni mejl.
