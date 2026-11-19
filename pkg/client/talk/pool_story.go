package talk

var storyLines = []Line{
	{Text: "Give me a moment, I'm still being kindled.", When: SitKindling, Tone: AnyTone},
	{Text: "Where's the quay? Which way is the quay?", When: SitKindling, Tone: Anxious, Needs: []string{"quay"}},
	{Text: "New hut, new me. Same boxes, I bet.", When: SitKindling, Tone: Joker},
	{Text: "Kindling! Ooh, it tickles.", When: SitKindling, Tone: Cheerful},
	{Text: "Hmph. Kindling. Nobody asked me.", When: SitKindling, Tone: Grumpy},
	{Text: "I shall be magnificent once I am lit.", When: SitKindling, Tone: Proud},
	{Text: "Sparks everywhere. Like little stars.", When: SitKindling, Tone: Dreamy},

	{Text: "Burning bright! That's the good kind of burning.", When: SitBurning, Tone: Cheerful},
	{Text: "Box in, box out. That's the whole song.", When: SitBurning, Tone: Grumpy, Pool: PoolBoth},
	{Text: "I carry, therefore I am.", When: SitBurning, Tone: Dreamy, Pool: PoolBoth},
	{Text: "Mind the step. There is no step.", When: SitBurning, Tone: Joker, Pool: PoolBoth, Then: "Mind it anyway."},
	{Text: "Is this right? I think this is right.", When: SitBurning, Tone: Anxious, Pool: PoolBoth},
	{Text: "Finest {trade} on the quay, if I say so myself.", When: SitBurning, Tone: Proud, Needs: []string{"quay"}},

	{Text: "Somewhere a box is waiting for me. I can feel it.", When: SitWaiting, Tone: Dreamy, Pool: PoolBoth},
	{Text: "No ships? Then I'll just stand here heroically.", When: SitWaiting, Tone: Proud, Pool: PoolBoth},
	{Text: "If the ships stop, do I stop?", When: SitWaiting, Tone: Anxious, Pool: PoolBoth, Then: "Please don't let them stop."},
	{Text: "Boxes, boxes, wherever are you, boxes?", When: SitWaiting, Tone: Cheerful, Pool: PoolBoth},
	{Text: "Nothing to carry. My arms forgot what for.", When: SitWaiting, Tone: Grumpy, Pool: PoolBoth},
	{Text: "I'm ready. The boxes are not. Their loss.", When: SitWaiting, Tone: Joker, Pool: PoolBoth},

	{Text: "Is there more bread? There's never more bread.", When: SitHungry, Tone: Grumpy},
	{Text: "I'd carry faster on a full stomach. Just saying.", When: SitHungry, Tone: Joker},
	{Text: "{rations} rations pledged, and I'm still peckish.", When: SitHungry, Tone: Proud, Needs: []string{"pledge", "rations"}, If: pledged},
	{Text: "So hungry. Everything is slow. Even my thoughts.", When: SitHungry, Tone: Dreamy},
	{Text: "My tummy rumbles louder than the cart.", When: SitHungry, Tone: Cheerful},
	{Text: "Hungry. Is that bad? That's bad, isn't it?", When: SitHungry, Tone: Anxious},

	{Text: "No room to turn round in here. Mind my elbows.", When: SitCramped, Tone: Grumpy},
	{Text: "Cramped! If I grow any more, the Void takes me.", When: SitCramped, Tone: Anxious, Needs: []string{"binding"}},
	{Text: "Rooms, rooms, I need more rooms!", When: SitCramped, Tone: Cheerful},
	{Text: "I fill my binding like a king fills a throne.", When: SitCramped, Tone: Proud, Needs: []string{"binding"}},
	{Text: "If I breathe in, the walls breathe in too.", When: SitCramped, Tone: Joker},
	{Text: "So many rooms, and all of them full of me.", When: SitCramped, Tone: Dreamy},

	{Text: "I'll get it right this time.", When: SitGuttering, Tone: Anxious, Then: "I said that last time."},
	{Text: "Stop, relight. Stop, relight. Stop, relight.", When: SitGuttering, Tone: Grumpy},
	{Text: "Don't watch. I'm about to stop again.", When: SitGuttering, Tone: Joker},
	{Text: "Up again! Up again! Ow. Up again!", When: SitGuttering, Tone: Cheerful},
	{Text: "A true {trade} stops only to be relit.", When: SitGuttering, Tone: Proud},
	{Text: "Down and up and down. Like waves.", When: SitGuttering, Tone: Dreamy},

	{Text: "The Void took me once. It was very roomy in there.", When: SitVoidtaken, Tone: Dreamy, Needs: []string{"voidtaken"}},
	{Text: "Too big for my binding. Happens to the best of us.", When: SitVoidtaken, Tone: Proud, Needs: []string{"binding"}},
	{Text: "I ate one room too many. Never again. Probably.", When: SitVoidtaken, Tone: Joker},
	{Text: "The Void! I saw the Void! Are we safe now?", When: SitVoidtaken, Tone: Anxious, Needs: []string{"voidtaken"}},
	{Text: "Voidtaken and back. I'm fine. FINE.", When: SitVoidtaken, Tone: Grumpy, Needs: []string{"voidtaken"}},
	{Text: "Back from the Void, and hungry for boxes!", When: SitVoidtaken, Tone: Cheerful, Needs: []string{"voidtaken"}},

	{Text: "The Academy never sent me. Or did it? I'm not here.", When: SitNever, Tone: AnyTone},

	{Text: "Tell the new class I left the quay tidy.", When: SitRites, Tone: Proud, Needs: []string{"quay"}, If: retiring},
	{Text: "My shift's over. All of it, it seems.", When: SitRites, Tone: Dreamy, Needs: []string{"crew"}, If: retiring},
	{Text: "Wave as I go? No? Fair.", When: SitRites, Tone: Joker},
	{Text: "Our hut is at Last Rites. I'll finish this box first.", When: SitRites, Tone: Proud, If: busy},
	{Text: "Last Rites for our hut. It was a good hut.", When: SitRites, Tone: Dreamy},
	{Text: "Last Rites already? I only just got here!", When: SitRites, Tone: Anxious, If: shortlived},
	{Text: "Last Rites. Fine. Somebody else can carry it.", When: SitRites, Tone: Grumpy},
	{Text: "Our hut's going. It's been a lovely day.", When: SitRites, Tone: Cheerful},

	{Text: "Ring the bell if I stop. I won't notice I stopped.", When: SitFrozen, Tone: Joker, Needs: []string{"bell"}, If: func(f Facts) bool { return f.Worker.Liveness }},
	{Text: "...", When: SitFrozen, Tone: AnyTone, Pool: PoolBoth},

	{Text: "Shutters down! Not open yet, come back soon.", When: SitShutters, Tone: Cheerful, Needs: []string{"shutters"}},
	{Text: "Still putting my boots on.", When: SitShutters, Tone: AnyTone, Pool: PoolBoth},
	{Text: "Shutters down. Do not rush me.", When: SitShutters, Tone: Grumpy, Needs: []string{"shutters"}},

	{Text: "Relit! Did anyone see me stop? No? Good.", When: SitRelit, Tone: Joker},
	{Text: "Relit and ready. Where were we?", When: SitRelit, Tone: Cheerful},
	{Text: "The Reeve relit me. I could have done it myself.", When: SitRelit, Tone: Proud, Needs: []string{"keep"}},
	{Text: "Relit. My head is still spinning.", When: SitRelit, Tone: Dreamy},
	{Text: "Relit again. Will it hold this time?", When: SitRelit, Tone: Anxious, If: again},
	{Text: "Relit. Don't make a fuss.", When: SitRelit, Tone: Grumpy},

	{Text: "Our hut is on the roll! Send the boxes!", When: SitJoined, Tone: Cheerful, Needs: []string{"roll"}},
	{Text: "On the roll at last. About time.", When: SitJoined, Tone: Grumpy, Needs: []string{"roll"}},
	{Text: "The roll has our hut's name on it. Fancy.", When: SitJoined, Tone: Proud, Needs: []string{"roll"}},
	{Text: "Our hut left the roll. No boxes for us now.", When: SitLeft, Tone: AnyTone, Needs: []string{"roll"}},

	{Text: "New class, same boxes.", When: SitRollout, Tone: Grumpy, Needs: []string{"crew"}},
	{Text: "The new class looks keen.", When: SitRollout, Tone: Joker, Needs: []string{"crew"}, If: retiring, Then: "I'm not jealous. A little."},
	{Text: "A new class is taking over, hut by hut.", When: SitRollout, Tone: Dreamy, Needs: []string{"crew"}},
	{Text: "New class? Will they need us old ones?", When: SitRollout, Tone: Anxious, Needs: []string{"crew"}, If: retiring},

	{Text: "Boxes pile up and there aren't enough of us.", When: SitProblem, Tone: AnyTone, Pool: PoolBoth, If: problem("spoiling")},
	{Text: "Some of our new huts have no Keep to stand on.", When: SitProblem, Tone: AnyTone, Needs: []string{"keep"}, If: problem("unhoused")},
	{Text: "The Keep's larder runs thin. We all eat less.", When: SitProblem, Tone: AnyTone, Needs: []string{"rations"}, If: problem("drought")},
	{Text: "The quay is full. Ships turn boxes away.", When: SitProblem, Tone: AnyTone, If: problem("quayfull")},

	{Text: "Did you see the Imp? Little thing. Big grudge.", When: SitImp, Tone: Joker, Needs: []string{"imp"}},
	{Text: "The Imp toppled a hut. Its shift raised a new one.", When: SitImp, Tone: Proud, Needs: []string{"imp", "crew"}, If: impReplaced},
	{Text: "The Imp is about. Keep your boxes close.", When: SitImp, Tone: Anxious, Needs: []string{"imp"}},

	{Text: "Gold on the water! Run!", When: SitGilded, Tone: Cheerful},
	{Text: "Gilded boxes carry themselves.", When: SitGilded, Tone: Joker, Then: "They don't. I wish."},

	{Text: "Here comes the tide! Everybody carry!", When: SitTide, Tone: Cheerful, Needs: []string{"tide"}},
	{Text: "So. Many. Ships.", When: SitTide, Tone: AnyTone, Pool: PoolBoth},

	{Text: "The lanterns are lit. The boxes don't care.", When: SitNight, Tone: Grumpy},
	{Text: "Night work. Same boxes, by lantern light.", When: SitNight, Tone: Dreamy},
	{Text: "Who carries boxes in the dark? Me, apparently.", When: SitNight, Tone: Joker, If: busy},

	{Text: "I answer to the Reeve and the door. Nobody else.", When: SitDoor, Tone: Proud, Needs: []string{"keep", "door"}},
	{Text: "No crew, no shift. Just me and the door.", When: SitDoor, Tone: Dreamy, Needs: []string{"crew", "door"}},

	{Text: "Nobody keeps my hut. I keep it myself.", When: SitHand, Tone: Proud},
	{Text: "If my hut is razed, nobody raises another.", When: SitHand, Tone: Grumpy, Then: "Lovely."},

	{Text: "The shift counts our huts all day long.", When: SitCrew, Tone: Cheerful, Needs: []string{"crew"}, Then: "Ours says here."},
	{Text: "If our hut is razed, the shift raises another.", When: SitCrew, Tone: Anxious, Needs: []string{"crew"}, Then: "Comforting?"},
	{Text: "Our hut's badge says {app}. Very official.", When: SitCrew, Tone: Proud, Needs: []string{"badge"}, If: hasApp},

	{Text: "Is our hut on the roll? It'd better be on the roll.", When: SitRoll, Tone: Anxious, Needs: []string{"roll"}},
	{Text: "The Stewards keep the roll. They never misspell our hut.", When: SitRoll, Tone: Joker, Needs: []string{"roll", "stewards"}},

	{Text: "The Citadel lives here. We tiptoe.", When: SitWard, Tone: AnyTone, Needs: []string{"citadel"}},

	{Text: "Relit before, and I'd do it again.", When: SitShaken, Tone: Proud},
	{Text: "I've stopped before. I'll stop again. Such is life.", When: SitShaken, Tone: Dreamy},

	{Text: "Porters don't walk.", When: SitTrade, Trade: "Porter", Tone: Joker, Then: "We glide. With a limp."},
	{Text: "Parcels today, parcels tomorrow, parcels forever.", When: SitTrade, Trade: "Porter", Tone: Grumpy},
	{Text: "Ledgers are just boxes with opinions.", When: SitTrade, Trade: "Clerk", Tone: Joker},
	{Text: "Seven columns, no mistakes.", When: SitTrade, Trade: "Clerk", Tone: Proud, Then: "Well. Six columns."},
	{Text: "Crates. Big crates. Always the big crates.", When: SitTrade, Trade: "Hauler", Tone: Grumpy},
	{Text: "Lift with the knees, they said.", When: SitTrade, Trade: "Hauler", Tone: Joker, Then: "I have one good knee."},
	{Text: "Faster than a gilded ship. Slower than gossip.", When: SitTrade, Trade: "Courier", Tone: Proud},
	{Text: "Every seal is a promise. I am bad at promises.", When: SitTrade, Trade: "Sealwright", Tone: Anxious},

	{Text: "You clicked me! Nobody ever clicks me.", When: SitGeneral, Tone: Cheerful, Pool: PoolBoth, If: firstClick},
	{Text: "Please stop poking. I'm working. Mostly.", When: SitGeneral, Tone: Grumpy, Pool: PoolBoth, If: Facts.Burning},
	{Text: "Looking for whoever's in charge? That's you.", When: SitGeneral, Tone: Joker, Pool: PoolBoth},
	{Text: "I counted the boxes once.", When: SitGeneral, Tone: Grumpy, Pool: PoolBoth, Then: "Never again."},
	{Text: "Every box is someone's wish. Heavy wishes today.", When: SitGeneral, Tone: Dreamy, Pool: PoolBoth, If: busy},
	{Text: "I daydreamed I was a crate. Very peaceful.", When: SitGeneral, Tone: Dreamy, Pool: PoolBoth},
	{Text: "One day I'll carry a gilded box. One day.", When: SitGeneral, Tone: Dreamy},
	{Text: "The Quartermaster put our hut here.", When: SitGeneral, Tone: Grumpy, Needs: []string{"quartermaster"}, If: scheduled, Then: "I wanted the sea view."},
	{Text: "When you speak at the Stone, we hear it in our bones.", When: SitGeneral, Tone: Dreamy, Needs: []string{"stone"}},
	{Text: "Old Borg never answers a question.", When: SitGeneral, Tone: Proud, Needs: []string{"borg"}, Then: "Just asks better ones."},
	{Text: "Learned a new page? I hope it's about parcels.", When: SitGeneral, Tone: Cheerful, Needs: []string{"grimoire"}},
	{Text: "Keeps cost gold now? Then I'll carry like a Keep does.", When: SitGeneral, Tone: Proud, Needs: []string{"keeps"}},
	{Text: "A new Age! Do we get new hats?", When: SitGeneral, Tone: Cheerful, Needs: []string{"ages"}, If: newAge},
	{Text: "What's the worst that could happen? Don't answer.", When: SitGeneral, Tone: Anxious, Pool: PoolBoth},
	{Text: "Hello! Lovely day for boxes.", When: SitGeneral, Tone: Cheerful, Pool: PoolBoth, If: daytime},
	{Text: "I do the work. You do the clicking. Fair.", When: SitGeneral, Tone: Proud, Pool: PoolBoth},
}

var petLines = []Line{
	{Text: "Kraa! Kraa-kraa.", When: SitPet, Tone: AnyTone, Pool: PoolBoth, If: pet("raven")},
	{Text: "Kraa. (It flies each chronicle line to the rookery.)", When: SitPet, Tone: AnyTone, Needs: []string{"chronicle"}, If: pet("raven")},
	{Text: "Kra. (It settles on the eaves.)", When: SitPet, Tone: AnyTone, If: pet("raven")},
	{Text: "Kraa. (A sidecar: it ships this Pod's logs.)", When: SitPet, Tone: AnyTone, Pool: PoolNames, If: pet("raven")},
	{Text: "Hrrmm. Hrrmmm.", When: SitPet, Tone: AnyTone, Pool: PoolBoth, If: pet("hound")},
	{Text: "Wuff. (It meets every box at the door.)", When: SitPet, Tone: AnyTone, If: pet("hound")},
	{Text: "Hrmf. (It leads a box to its master.)", When: SitPet, Tone: AnyTone, If: pet("hound")},
	{Text: "Wuff. (A sidecar proxy: each request passes it.)", When: SitPet, Tone: AnyTone, Pool: PoolNames, If: pet("hound")},
}

var annoyedLines = []Line{
	{Text: "Yes? Still here. Still working.", When: SitAnnoyed, Tone: AnyTone, Pool: PoolBoth},
	{Text: "Do you do this to the Keeps too?", When: SitAnnoyed, Tone: AnyTone, Needs: []string{"keep"}},
	{Text: "Do you do this to the Nodes too?", When: SitAnnoyed, Tone: AnyTone, Pool: PoolNames, Needs: []string{"keep"}},
	{Text: "I'm going to start charging for this.", When: SitAnnoyed, Tone: AnyTone, Pool: PoolBoth},
	{Text: "Fine. FINE. I'm counting the pokes now.", When: SitAnnoyed, Tone: AnyTone, Pool: PoolBoth},
	{Text: "That's it. I'm telling the Reeve.", When: SitAnnoyed, Tone: AnyTone, Needs: []string{"keep"}},
	{Text: "Is this a liveness probe? I'm alive. Promise.", When: SitAnnoyed, Tone: AnyTone, Pool: PoolNames, Needs: []string{"bell"}},
}

func problem(key string) func(Facts) bool {
	return func(f Facts) bool { return f.Problem == key }
}

func pet(kind string) func(Facts) bool {
	return func(f Facts) bool { return f.Familiar == kind }
}
